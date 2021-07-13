package jwtclient

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/venus-auth/auth"
	"github.com/filecoin-project/venus-auth/errcode"
	"github.com/go-resty/resty/v2"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	"net/http"
	"strconv"
)

type JWTClient struct {
	cli *resty.Client
}

func NewJWTClient(url string) *JWTClient {
	client := resty.New().
		SetHostURL(url).
		SetRetryCount(2).
		SetHeader("Accept", "application/json").
		SetTransport(&ochttp.Transport{})

	return &JWTClient{
		cli: client,
	}
}

// Verify: post method for Verify token
// @spanId: local service unique Id
// @serviceName: e.g. venus
// @preHost: the IP of the request server
// @host: local service IP
// @token: jwt token gen from this service
func (c *JWTClient) Verify(ctx context.Context, token string) (*auth.VerifyResponse, error) {
	ctx, span := trace.StartSpan(ctx, "JWTClient.verify",
		func(so *trace.StartOptions) { so.Sampler = trace.AlwaysSample() })
	defer span.End()

	req := c.cli.R().SetContext(ctx).
		SetFormData(map[string]string{"token": token})

	response, err := req.Post("/verify")

	if err != nil {
		return nil, err
	}
	switch response.StatusCode() {
	case http.StatusOK:
		var res = new(auth.VerifyResponse)
		err = json.Unmarshal(response.Body(), res)
		span.AddAttributes(trace.StringAttribute("Account", res.Name))
		return res, err
	default:
		response.Result()
		span.SetStatus(trace.Status{
			Code:    trace.StatusCodeUnauthenticated,
			Message: string(response.Body()),
		})
		return nil, fmt.Errorf("response code is : %d, msg:%s", response.StatusCode(), response.Body())
	}
}

func (c *JWTClient) ListAccounts(req *auth.ListAccountsRequest) (auth.ListAccountsResponse, error) {
	resp, err := c.cli.R().SetQueryParams(map[string]string{
		"skip":       strconv.FormatInt(req.Skip, 10),
		"limit":      strconv.FormatInt(req.Limit, 10),
		"sourceType": strconv.Itoa(req.SourceType),
		"state":      strconv.Itoa(req.State),
		"keySum":     strconv.Itoa(req.KeySum),
	}).SetResult(&auth.ListAccountsResponse{}).SetError(&errcode.ErrMsg{}).Get("/account/list")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusOK {
		return *(resp.Result().(*auth.ListAccountsResponse)), nil
	}
	return nil, resp.Error().(*errcode.ErrMsg).Err()
}

func (c *JWTClient) GetAccount(req *auth.GetAccountRequest) (*auth.OutputAccount, error) {
	resp, err := c.cli.R().SetQueryParams(map[string]string{
		"name": req.Name,
	}).SetResult(&auth.OutputAccount{}).SetError(&errcode.ErrMsg{}).Get("/account")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusOK {
		return resp.Result().(*auth.OutputAccount), nil
	}
	return nil, resp.Error().(*errcode.ErrMsg).Err()
}

func (c *JWTClient) GetMiner(req *auth.GetMinerRequest) (*auth.OutputAccount, error) {
	resp, err := c.cli.R().SetQueryParams(map[string]string{
		"miner": req.Miner,
	}).SetResult(&auth.OutputAccount{}).SetError(&errcode.ErrMsg{}).Get("/miner")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusOK {
		return resp.Result().(*auth.OutputAccount), nil
	}
	return nil, resp.Error().(*errcode.ErrMsg).Err()
}

func (c *JWTClient) HasMiner(req *auth.HasMinerRequest) (bool, error) {
	var has bool
	resp, err := c.cli.R().SetQueryParams(map[string]string{
		"miner": req.Miner,
	}).SetResult(&has).SetError(&errcode.ErrMsg{}).Get("/miner/has-miner")
	if err != nil {
		return false, err
	}
	if resp.StatusCode() == http.StatusOK {
		return *resp.Result().(*bool), nil
	}
	return false, resp.Error().(*errcode.ErrMsg).Err()
}
