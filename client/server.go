package client

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"google.golang.org/grpc"
)

type TaskCallback func(resp *clientpb.TaskContext)

func NewServerStatus(conn *grpc.ClientConn, config *mtls.ClientConfig) (*ServerState, error) {
	var err error
	s := &ServerState{
		Rpc: &Rpc{
			MaliceRPCClient:   clientrpc.NewMaliceRPCClient(conn),
			ListenerRPCClient: listenerrpc.NewListenerRPCClient(conn),
		},
		ActiveTarget:    &ActiveTarget{},
		Listeners:       make(map[string]*clientpb.Listener),
		Pipelines:       make(map[string]*clientpb.Pipeline),
		Sessions:        make(map[string]*Session),
		Observers:       make(map[string]*Session),
		FinishCallbacks: &sync.Map{},
		DoneCallbacks:   &sync.Map{},
		EventHook:       make(map[EventCondition][]OnEventFunc),
		EventCallback:   make(map[string]func(*clientpb.Event)),
	}
	client, err := s.Rpc.LoginClient(context.Background(), &clientpb.LoginReq{
		Name: config.Operator,
		Host: config.Host,
		Port: uint32(config.Port),
	})
	if err != nil {
		return nil, err
	}
	s.Client = client
	s.Info, err = s.Rpc.GetBasic(context.Background(), &clientpb.Empty{})
	if err != nil {
		return nil, err
	}

	err = s.Update()
	if err != nil {
		return nil, err
	}
	return s, nil
}

type Rpc struct {
	clientrpc.MaliceRPCClient
	listenerrpc.ListenerRPCClient
}

type ServerState struct {
	*Rpc
	Info   *clientpb.Basic
	Client *clientpb.Client
	*ActiveTarget
	Clients         []*clientpb.Client
	Listeners       map[string]*clientpb.Listener
	Pipelines       map[string]*clientpb.Pipeline
	Sessions        map[string]*Session
	Observers       map[string]*Session
	mu              sync.RWMutex
	FinishCallbacks *sync.Map
	DoneCallbacks   *sync.Map
	EventStatus     bool
	EventHook       map[EventCondition][]OnEventFunc
	EventCallback   map[string]func(*clientpb.Event)
}

// ReconcileEvent is the single entry point for updating client-side state
// from server events. All map mutations go through here.
func (s *ServerState) ReconcileEvent(event *clientpb.Event) {
	switch event.Type {
	case consts.EventSession:
		s.reconcileSession(event)
	case consts.EventJob:
		s.reconcilePipeline(event)
	case consts.EventListener:
		s.reconcileListener(event)
	case consts.EventClient:
		s.reconcileClient(event)
	}
}

func (s *ServerState) reconcileSession(event *clientpb.Event) {
	if event.Session == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch event.Op {
	case consts.CtrlSessionRegister, consts.CtrlSessionInit,
		consts.CtrlSessionReborn, consts.CtrlSessionUpdate, consts.CtrlSessionCheckin:
		s.addSessionLocked(event.Session)
	case consts.CtrlSessionDead:
		sid := event.Session.SessionId
		delete(s.Sessions, sid)
		if s.ActiveTarget != nil && s.Session != nil && s.Session.SessionId == sid {
			s.Background()
		}
	}
}

func (s *ServerState) reconcilePipeline(event *clientpb.Event) {
	job := event.GetJob()
	pipeline := job.GetPipeline()
	if pipeline == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch event.Op {
	case consts.CtrlPipelineSync, consts.CtrlPipelineStart, consts.CtrlWebsiteStart, consts.CtrlRemStart:
		s.Pipelines[pipeline.Name] = pipeline
	case consts.CtrlPipelineStop, consts.CtrlWebsiteStop, consts.CtrlRemStop:
		delete(s.Pipelines, pipeline.Name)
	case consts.CtrlWebContentAdd, consts.CtrlWebContentAddArtifact:
		current, ok := s.Pipelines[pipeline.Name]
		if !ok || current == nil {
			s.Pipelines[pipeline.Name] = pipeline
			current = pipeline
		}
		if current.GetWeb() == nil {
			s.Pipelines[pipeline.Name] = pipeline
			return
		}
		if current.GetWeb().Contents == nil {
			current.GetWeb().Contents = make(map[string]*clientpb.WebContent)
		}
		for path, content := range pipeline.GetWeb().GetContents() {
			if content == nil {
				continue
			}
			if path == "" {
				path = content.Path
			}
			current.GetWeb().Contents[path] = content
		}
		for path, content := range job.GetContents() {
			if content == nil {
				continue
			}
			if path == "" {
				path = content.Path
			}
			current.GetWeb().Contents[path] = content
		}
	case consts.CtrlWebContentRemove:
		current, ok := s.Pipelines[pipeline.Name]
		if !ok || current == nil || current.GetWeb() == nil || current.GetWeb().Contents == nil {
			return
		}
		for path, content := range pipeline.GetWeb().GetContents() {
			if path == "" && content != nil {
				path = content.Path
			}
			if path != "" {
				delete(current.GetWeb().Contents, path)
			}
		}
		for path, content := range job.GetContents() {
			if path == "" && content != nil {
				path = content.Path
			}
			if path != "" {
				delete(current.GetWeb().Contents, path)
			}
		}
	}
}

func (s *ServerState) reconcileListener(event *clientpb.Event) {
	listener := event.GetListener()
	if listener == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch event.Op {
	case consts.CtrlListenerStart:
		s.Listeners[listener.Id] = listener
	case consts.CtrlListenerStop:
		delete(s.Listeners, listener.Id)
	}
}

func (s *ServerState) reconcileClient(event *clientpb.Event) {
	cl := event.GetClient()
	if cl == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch event.Op {
	case consts.CtrlClientJoin:
		s.Clients = append(s.Clients, cl)
	case consts.CtrlClientLeft:
		for i, c := range s.Clients {
			if c.ID == cl.ID {
				s.Clients = append(s.Clients[:i], s.Clients[i+1:]...)
				break
			}
		}
	}
}

func (s *ServerState) Update() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	clients, err := s.Rpc.GetClients(context.Background(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	s.Clients = clients.GetClients()

	err = s.updateListenerLocked()
	if err != nil {
		return err
	}

	err = s.updatePipelineLocked()
	if err != nil {
		return err
	}

	err = s.updateSessionsLocked(false)
	if err != nil {
		return err
	}
	return nil
}

// addSessionLocked adds or updates a session. Caller must hold s.mu.
func (s *ServerState) addSessionLocked(sess *clientpb.Session) *Session {
	if origin, ok := s.Sessions[sess.SessionId]; ok {
		origin.Session = sess
		return origin
	}
	s.Sessions[sess.SessionId] = NewSession(sess, s)
	return s.Sessions[sess.SessionId]
}

func (s *ServerState) AddSession(sess *clientpb.Session) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.addSessionLocked(sess)
}

func (s *ServerState) UpdateSessions(all bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateSessionsLocked(all)
}

func (s *ServerState) updateSessionsLocked(all bool) error {
	if s == nil {
		return errors.New("You need login first")
	}
	sessions, err := s.Rpc.GetSessions(context.Background(), &clientpb.SessionRequest{
		All: all,
	})
	if err != nil {
		return err
	}
	newSessions := make(map[string]*Session)

	for _, session := range sessions.GetSessions() {
		if rawSess, ok := s.Sessions[session.SessionId]; ok {
			rawSess.Session = session
			newSessions[session.SessionId] = rawSess
		} else {
			newSessions[session.SessionId] = NewSession(session, s)
		}
	}

	s.Sessions = newSessions
	return nil
}

func (s *ServerState) UpdateSession(sid string) (*Session, error) {
	session, err := s.Rpc.GetSession(context.Background(), &clientpb.SessionRequest{SessionId: sid})
	if err != nil {
		return nil, err
	}
	if session == nil || session.SessionId == "" {
		return nil, fmt.Errorf("session %s not found", sid)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if rawSess, ok := s.Sessions[session.SessionId]; ok {
		rawSess.Session = session
		return rawSess, nil
	}
	newSess := NewSession(session, s)
	s.Sessions[session.SessionId] = newSess
	return newSess, nil
}

func (s *ServerState) GetLocalSession(sid string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if sess, ok := s.Sessions[sid]; ok {
		return sess, true
	}
	return nil, false
}

func (s *ServerState) GetOrUpdateSession(sid string) (*Session, error) {
	s.mu.RLock()
	sess, ok := s.Sessions[sid]
	s.mu.RUnlock()
	if ok {
		return sess, nil
	}
	return s.UpdateSession(sid)
}

func (s *ServerState) AlivedSessions() []*clientpb.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var alivedSessions []*clientpb.Session
	for _, session := range s.Sessions {
		if session.IsAlive {
			alivedSessions = append(alivedSessions, session.Session)
		}
	}
	return alivedSessions
}

func (s *ServerState) UpdateTasks(session *Session) error {
	if session == nil {
		return errors.New("session is nil")
	}
	tasks, err := s.Rpc.GetTasks(context.Background(), &clientpb.TaskRequest{
		SessionId: session.SessionId,
	})
	if err != nil {
		return err
	}

	session.Tasks = &clientpb.Tasks{Tasks: tasks.Tasks}
	return nil
}

func (s *ServerState) UpdateListener() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateListenerLocked()
}

func (s *ServerState) updateListenerLocked() error {
	listeners, err := s.Rpc.GetListeners(context.Background(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	s.Listeners = make(map[string]*clientpb.Listener)
	for _, listener := range listeners.GetListeners() {
		s.Listeners[listener.Id] = listener
	}
	return nil
}

func (s *ServerState) UpdatePipeline() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updatePipelineLocked()
}

func (s *ServerState) updatePipelineLocked() error {
	pipelines, err := s.Rpc.ListPipelines(context.Background(), &clientpb.Listener{})
	if err != nil {
		return err
	}
	s.Pipelines = make(map[string]*clientpb.Pipeline)
	for _, pipeline := range pipelines.GetPipelines() {
		s.Pipelines[pipeline.Name] = pipeline
	}

	websites, err := s.Rpc.ListWebsites(context.Background(), &clientpb.Listener{})
	if err != nil {
		return err
	}
	for _, website := range websites.GetPipelines() {
		if web := website.GetWeb(); web != nil {
			contents, contentErr := s.Rpc.ListWebContent(context.Background(), &clientpb.Website{Name: website.Name})
			if contentErr != nil {
				return contentErr
			}
			web.Contents = make(map[string]*clientpb.WebContent, len(contents.GetContents()))
			for _, content := range contents.GetContents() {
				if content == nil {
					continue
				}
				web.Contents[content.Path] = content
			}
		}
		s.Pipelines[website.Name] = website
	}
	return nil
}

func (s *ServerState) AddObserver(session *Session) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	Log.Infof("Add observer to %s", session.SessionId)
	s.Observers[session.SessionId] = session
	return session.SessionId
}

func (s *ServerState) RemoveObserver(observerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Observers, observerID)
}

func (s *ServerState) ObserverLog(sessionId string) *Logger {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Session != nil && s.Session.SessionId == sessionId {
		return s.Session.Log
	}

	if observer, ok := s.Observers[sessionId]; ok {
		return observer.Log
	}
	return MuteLog
}
