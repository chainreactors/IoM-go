package clientpb

import (
	"encoding/binary"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
)

func tlsEnabled(pipe *Pipeline) bool {
	return pipe != nil && pipe.Tls != nil && pipe.Tls.Enable
}

func (pipe *Pipeline) Address() string {
	if pipe == nil {
		return ""
	}
	switch body := pipe.Body.(type) {
	case *Pipeline_Http:
		return fmt.Sprintf("%s:%d", pipe.Ip, body.Http.Port)
	case *Pipeline_Tcp:
		return fmt.Sprintf("%s:%d", pipe.Ip, body.Tcp.Port)
	case *Pipeline_Rem:
		return fmt.Sprintf("%s:%d", pipe.Ip, body.Rem.Port)
	default:
		return ""
	}
}

func (task *Task) Progress() string {
	if task.Total == -1 {
		return fmt.Sprintf("%d/∞", task.Cur)
	} else {
		return fmt.Sprintf("%d/%d", task.Cur, task.Total)
	}
}

func (pipe *Pipeline) URL() string {
	scheme := "http"
	if tlsEnabled(pipe) {
		scheme = "https"
	}
	if pipe == nil {
		return ""
	}

	if pipe.Type == consts.WebsitePipeline {
		web := pipe.GetWeb()
		if web == nil {
			return ""
		}
		// baseURL 只到 host:port
		return fmt.Sprintf("%s://%s:%d", scheme, pipe.Ip, web.Port) + web.Root
	} else if pipe.Type == consts.HTTPPipeline {
		if pipe.GetHttp() == nil {
			return ""
		}
		return fmt.Sprintf("%s://%s:%d", scheme, pipe.Ip, pipe.GetHttp().Port)
	} else if pipe.Type == consts.TCPPipeline {
		if pipe.GetTcp() == nil {
			return ""
		}
		return fmt.Sprintf("tcp://%s:%d", pipe.Ip, pipe.GetTcp().Port)
	}

	return ""
}

func (pipe *Job) FirstContent() *WebContent {
	for _, content := range pipe.Contents {
		return content
	}
	return nil
}

func (pipe *Pipeline) KVMap() (map[string]interface{}, []string) {
	pipelineMap := map[string]interface{}{
		"Name":        pipe.Name,
		"Type":        pipe.Type,
		"Listener ID": pipe.ListenerId,
	}

	var orderedKeys []string
	orderedKeys = append(orderedKeys, "Name", "Type", "Listener ID")

	switch pipe.Body.(type) {
	case *Pipeline_Tcp:
		pipelineMap["Address"] = pipe.Address()
		pipelineMap["TLS"] = tlsEnabled(pipe)
		pipelineMap["Cert"] = pipe.CertName
		orderedKeys = append(orderedKeys, "Address", "TLS", "Cert")
	case *Pipeline_Http:
		pipelineMap["Address"] = pipe.Address()
		pipelineMap["TLS"] = tlsEnabled(pipe)
		pipelineMap["Cert"] = pipe.CertName
		orderedKeys = append(orderedKeys, "Address", "TLS", "Cert")
	case *Pipeline_Bind:
		pipelineMap["Ip"] = pipe.Ip
		orderedKeys = append(orderedKeys, "Ip")
	case *Pipeline_Rem:
		pipelineMap["Address"] = pipe.Address()
		orderedKeys = append(orderedKeys, "Address")
	case *Pipeline_Web:
		pipelineMap["Port"] = pipe.GetWeb().Port
		pipelineMap["URL"] = pipe.URL()
		pipelineMap["TLS"] = tlsEnabled(pipe)
		pipelineMap["Cert"] = pipe.CertName
		orderedKeys = append(orderedKeys, "Port", "URL", "TLS", "Cert")
	case *Pipeline_Custom:
		if c := pipe.GetCustom(); c != nil {
			if c.Host != "" {
				pipelineMap["Host"] = c.Host
				orderedKeys = append(orderedKeys, "Host")
			}
			if c.Port > 0 {
				pipelineMap["Port"] = c.Port
				orderedKeys = append(orderedKeys, "Port")
			}
		}
	}

	return pipelineMap, orderedKeys
}

func (session *Session) Raw() []byte {
	raw := make([]byte, 4)
	binary.LittleEndian.PutUint32(raw, session.RawId)
	return raw
}
