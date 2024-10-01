package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-retryablehttp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	rpcapi "github.com/spacemeshos/poet/release/proto/go/rpc/api/v1"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrUnavailable    = errors.New("unavailable")
	ErrInvalidRequest = errors.New("invalid request")
)

type PoetPowParams struct {
	Challenge  []byte
	Difficulty uint
}

type PoetPoW struct {
	Nonce  uint64
	Params PoetPowParams
}

type CertifierInfo struct {
	URL    *url.URL
	PubKey []byte
}

// HTTPPoetClient implements PoetProvingServiceClient interface.
type HTTPPoetClient struct {
	baseURL *url.URL
	client  *retryablehttp.Client
}

// NewHTTPPoetClient returns new instance of HTTPPoetClient connecting to the specified url.
func NewHTTPPoetClient(baseUrl string) (*HTTPPoetClient, error) {
	baseURL, err := url.Parse(baseUrl)
	if err != nil {
		return nil, fmt.Errorf("parsing address: %w", err)
	}
	if baseURL.Scheme == "" {
		baseURL.Scheme = "http"
	}

	return &HTTPPoetClient{
		baseURL: baseURL,
		client:  retryablehttp.NewClient(),
	}, nil
}

func (c *HTTPPoetClient) info(ctx context.Context) (*rpcapi.InfoResponse, error) {
	resBody := rpcapi.InfoResponse{}
	if err := c.req(ctx, http.MethodGet, "/v1/info", nil, &resBody); err != nil {
		return nil, fmt.Errorf("getting poet ID: %w", err)
	}
	return &resBody, nil
}

func (c *HTTPPoetClient) CertifierInfo(ctx context.Context) (*CertifierInfo, error) {
	// info, err := c.info(ctx)
	// if err != nil {
	// 	return nil, err
	// }
	// // certifierInfo := info.GetCertifier()
	// if certifierInfo == nil {
	// 	return nil, errors.New("poet doesn't support certifier")
	// }
	// url, err := url.Parse(certifierInfo.Url)
	// if err != nil {
	// 	return nil, fmt.Errorf("parsing certifier address: %w", err)
	// }
	// return &CertifierInfo{
	// 	PubKey: certifierInfo.Pubkey,
	// 	URL:    url,
	// }, nil
	return nil, nil
}

func (c *HTTPPoetClient) PowParams(ctx context.Context) (*PoetPowParams, error) {
	resBody := rpcapi.PowParamsResponse{}
	if err := c.req(ctx, http.MethodGet, "/v1/pow_params", nil, &resBody); err != nil {
		return nil, fmt.Errorf("querying PoW params: %w", err)
	}

	return &PoetPowParams{
		Challenge:  resBody.GetPowParams().GetChallenge(),
		Difficulty: uint(resBody.GetPowParams().GetDifficulty()),
	}, nil
}

// Submit registers a challenge in the proving service current open round.
func (c *HTTPPoetClient) Submit(
	ctx context.Context,
	prefix, challenge, signature, nodeID []byte,
	pow PoetPoW,
	cert []byte,
) error {
	request := rpcapi.SubmitRequest{
		Prefix:    prefix,
		Challenge: challenge,
		Signature: signature,
		Pubkey:    nodeID,
		Nonce:     pow.Nonce,
		PowParams: &rpcapi.PowParams{
			Challenge:  pow.Params.Challenge,
			Difficulty: uint32(pow.Params.Difficulty),
		},
		// Certificate: &rpcapi.SubmitRequest_Certificate{
		// 	Signature: cert,
		// },
	}
	resBody := rpcapi.SubmitResponse{}
	return c.req(ctx, http.MethodPost, "/v1/submit", &request, &resBody)
}

// PoetServiceID returns the public key of the PoET proving service.
func (c *HTTPPoetClient) PoetServiceID(ctx context.Context) ([]byte, error) {
	resBody := rpcapi.InfoResponse{}

	if err := c.req(ctx, http.MethodGet, "/v1/info", nil, &resBody); err != nil {
		return nil, fmt.Errorf("getting poet ID: %w", err)
	}

	return resBody.ServicePubkey, nil
}

// Proof implements PoetProvingServiceClient.
func (c *HTTPPoetClient) Proof(ctx context.Context, roundID string) (*rpcapi.PoetProof, error) {
	resBody := rpcapi.ProofResponse{}
	if err := c.req(ctx, http.MethodGet, fmt.Sprintf("/v1/proofs/%s", roundID), nil, &resBody); err != nil {
		return nil, fmt.Errorf("getting proof: %w", err)
	}

	return resBody.Proof, nil
}

func (c *HTTPPoetClient) req(ctx context.Context, method, path string, reqBody, resBody proto.Message) error {
	jsonReqBody, err := protojson.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshaling request body: %w", err)
	}

	req, err := retryablehttp.NewRequestWithContext(
		ctx,
		method,
		c.baseURL.JoinPath(path).String(),
		bytes.NewReader(jsonReqBody),
	)
	if err != nil {
		return fmt.Errorf("creating HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("doing request: %w", err)
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body (%w)", err)
	}

	fmt.Printf("response size: %.2f MiB\n", float64(len(data))/1024/1024)

	switch res.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return fmt.Errorf("%w: response status code: %s, body: %s", ErrNotFound, res.Status, string(data))
	case http.StatusServiceUnavailable:
		return fmt.Errorf("%w: response status code: %s, body: %s", ErrUnavailable, res.Status, string(data))
	case http.StatusBadRequest:
		return fmt.Errorf("%w: response status code: %s, body: %s", ErrInvalidRequest, res.Status, string(data))
	default:
		return fmt.Errorf("unrecognized error: status code: %s, body: %s", res.Status, string(data))
	}

	if resBody != nil {
		unmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}
		if err := unmarshaler.Unmarshal(data, resBody); err != nil {
			return fmt.Errorf("decoding response body to proto: %w", err)
		}
	}

	return nil
}
