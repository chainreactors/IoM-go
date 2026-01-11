package types

import (
	"errors"
	"fmt"

	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
)

var (
	ErrNilStatus       = errors.New("nil status or unknown error")
	ErrAssertFailure   = errors.New("assert spite type failure")
	ErrNilResponseBody = errors.New("must return spite body")
)

func HandleMaleficError(content *implantpb.Spite) error {
	if content == nil {
		return fmt.Errorf("nil spite")
	}
	var err error
	switch content.Error {
	case 0:
		return nil
	case MaleficErrorPanic:
		err = fmt.Errorf("module Panic")
	case MaleficErrorUnpackError:
		err = fmt.Errorf("module unpack error")
	case MaleficErrorMissbody:
		err = fmt.Errorf("module miss body")
	case MaleficErrorModuleError:
		err = fmt.Errorf("module error")
	case MaleficErrorModuleNotFound:
		err = fmt.Errorf("module not found")
	case MaleficErrorTaskError:
		return HandleTaskError(content.Status)
	case MaleficErrorTaskNotFound:
		err = fmt.Errorf("task not found")
	case MaleficErrorTaskOperatorNotFound:
		err = fmt.Errorf("task operator not found")
	case MaleficErrorExtensionNotFound:
		err = fmt.Errorf("extension not found")
	case MaleficErrorUnexceptBody:
		err = fmt.Errorf("unexcept body")
	default:
		err = fmt.Errorf("unknown Malefic error, %d", content.Error)
	}
	return err
}

func HandleTaskError(status *implantpb.Status) error {
	var err error
	switch status.Status {
	case 0:
		return nil
	case TaskErrorOperatorError:
		err = fmt.Errorf("task error: %s", status.Error)
	case TaskErrorNotExpectBody:
		err = fmt.Errorf("task error: %s", status.Error)
	case TaskErrorFieldRequired:
		err = fmt.Errorf("task error: %s", status.Error)
	case TaskErrorFieldLengthMismatch:
		err = fmt.Errorf("task error: %s", status.Error)
	case TaskErrorFieldInvalid:
		err = fmt.Errorf("task error: %s", status.Error)
	case TaskError:
		err = fmt.Errorf("task error: %s", status.Error)
	default:
		err = fmt.Errorf("unknown error, %v", status)
	}
	return err
}

func AssertRequestName(req *implantpb.Request, expect MsgName) error {
	if req.Name == "" {
		req.Name = expect.String()
	}

	if req.Name != string(expect) {
		return fmt.Errorf("%w, assert request name failure, expect %s, got %s", ErrAssertFailure, expect, req.Name)
	}
	return nil
}

func AssertSpite(spite *implantpb.Spite, expect MsgName) error {
	body := spite.GetBody()
	if body == nil && expect != MsgNil {
		return ErrNilResponseBody
	}
	if expect == "" {
		return nil
	}

	if expect != MessageType(spite) {
		return fmt.Errorf("%w, assert response type failure, expect %s, got %s", ErrAssertFailure, expect, MessageType(spite))
	}
	return nil
}

func AssertStatusAndSpite(spite *implantpb.Spite, expect MsgName) error {
	if err := HandleMaleficError(spite); err != nil {
		return err
	}
	return AssertSpite(spite, expect)
}
