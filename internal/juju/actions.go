package juju

import (
	"errors"

	"github.com/juju/juju/api"
	"github.com/juju/juju/api/client/action"
	"github.com/juju/names/v5"
)

type ActionResultArgs struct {
	ModelName string
	ActionId  string
}

type EnqueueActionArgs struct {
	ModelName string
	ActionQ   action.Action
}

type actionsClient struct {
	SharedClient

	getActionsAPIClient func(connection api.Connection) *action.Client
}

func newActionsClient(sc SharedClient) *actionsClient {
	return &actionsClient{
		SharedClient: sc,
		getActionsAPIClient: func(connection api.Connection) *action.Client {
			return action.NewClient(connection)
		},
	}
}

func (c *actionsClient) ActionResult(args ActionResultArgs) (action.ActionResult, error) {
	conn, err := c.GetConnection(&args.ModelName)
	if err != nil {
		return action.ActionResult{}, err
	}
	defer func() { _ = conn.Close() }()

	actionsAPIClient := c.getActionsAPIClient(conn)

	results, err := actionsAPIClient.Actions([]string{args.ActionId})
	if err != nil {
		return action.ActionResult{}, err
	}
	if len(results) != 1 {
		return action.ActionResult{}, errors.New("expected exactly one action result, got " + string(len(results)))
	}
	return results[0], nil
}

func (c *actionsClient) EnqueueAction(args EnqueueActionArgs) (string, error) {
	conn, err := c.GetConnection(&args.ModelName)
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()

	actionsAPIClient := c.getActionsAPIClient(conn)
	args.ActionQ.Receiver = names.NewUnitTag(args.ActionQ.Receiver).String()
	enqueuedActions, err := actionsAPIClient.EnqueueOperation([]action.Action{args.ActionQ})
	if err != nil {
		return "", err
	}
	if len(enqueuedActions.Actions) != 1 {
		return "", errors.New("expected exactly one enqueued action, got " + string(len(enqueuedActions.Actions)))
	}
	if enqueuedActions.Actions[0].Action == nil {
		return "", errors.New("enqueued action is nil")
	}
	return enqueuedActions.Actions[0].Action.ID, nil
}
