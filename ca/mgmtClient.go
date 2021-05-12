package ca

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"path"

	"github.com/pkg/errors"
	"github.com/smallstep/certificates/authority/mgmt"
	mgmtAPI "github.com/smallstep/certificates/authority/mgmt/api"
	"github.com/smallstep/certificates/errs"
)

// MgmtClient implements an HTTP client for the CA server.
type MgmtClient struct {
	client    *uaClient
	endpoint  *url.URL
	retryFunc RetryFunc
	opts      []ClientOption
}

// NewMgmtClient creates a new MgmtClient with the given endpoint and options.
func NewMgmtClient(endpoint string, opts ...ClientOption) (*MgmtClient, error) {
	u, err := parseEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	// Retrieve transport from options.
	o := new(clientOptions)
	if err := o.apply(opts); err != nil {
		return nil, err
	}
	tr, err := o.getTransport(endpoint)
	if err != nil {
		return nil, err
	}

	return &MgmtClient{
		client:    newClient(tr),
		endpoint:  u,
		retryFunc: o.retryFunc,
		opts:      opts,
	}, nil
}

func (c *MgmtClient) retryOnError(r *http.Response) bool {
	if c.retryFunc != nil {
		if c.retryFunc(r.StatusCode) {
			o := new(clientOptions)
			if err := o.apply(c.opts); err != nil {
				return false
			}
			tr, err := o.getTransport(c.endpoint.String())
			if err != nil {
				return false
			}
			r.Body.Close()
			c.client.SetTransport(tr)
			return true
		}
	}
	return false
}

// GetAdmin performs the GET /mgmt/admin/{id} request to the CA.
func (c *MgmtClient) GetAdmin(id string) (*mgmt.Admin, error) {
	var retried bool
	u := c.endpoint.ResolveReference(&url.URL{Path: path.Join("/mgmt/admin", id)})
retry:
	resp, err := c.client.Get(u.String())
	if err != nil {
		return nil, errors.Wrapf(err, "client GET %s failed", u)
	}
	if resp.StatusCode >= 400 {
		if !retried && c.retryOnError(resp) {
			retried = true
			goto retry
		}
		return nil, readError(resp.Body)
	}
	var adm = new(mgmt.Admin)
	if err := readJSON(resp.Body, adm); err != nil {
		return nil, errors.Wrapf(err, "error reading %s", u)
	}
	return adm, nil
}

// CreateAdmin performs the POST /mgmt/admin request to the CA.
func (c *MgmtClient) CreateAdmin(req *mgmtAPI.CreateAdminRequest) (*mgmt.Admin, error) {
	var retried bool
	body, err := json.Marshal(req)
	if err != nil {
		return nil, errs.Wrap(http.StatusInternalServerError, err, "error marshaling request")
	}
	u := c.endpoint.ResolveReference(&url.URL{Path: "/mgmt/admin"})
retry:
	resp, err := c.client.Post(u.String(), "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrapf(err, "client POST %s failed", u)
	}
	if resp.StatusCode >= 400 {
		if !retried && c.retryOnError(resp) {
			retried = true
			goto retry
		}
		return nil, readError(resp.Body)
	}
	var adm = new(mgmt.Admin)
	if err := readJSON(resp.Body, adm); err != nil {
		return nil, errors.Wrapf(err, "error reading %s", u)
	}
	return adm, nil
}

// RemoveAdmin performs the DELETE /mgmt/admin/{id} request to the CA.
func (c *MgmtClient) RemoveAdmin(id string) error {
	var retried bool
	u := c.endpoint.ResolveReference(&url.URL{Path: path.Join("/mgmt/admin", id)})
	req, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return errors.Wrapf(err, "create DELETE %s request failed", u)
	}
retry:
	resp, err := c.client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "client DELETE %s failed", u)
	}
	if resp.StatusCode >= 400 {
		if !retried && c.retryOnError(resp) {
			retried = true
			goto retry
		}
		return readError(resp.Body)
	}
	return nil
}

// UpdateAdmin performs the PUT /mgmt/admin/{id} request to the CA.
func (c *MgmtClient) UpdateAdmin(id string, uar *mgmtAPI.UpdateAdminRequest) (*mgmt.Admin, error) {
	var retried bool
	body, err := json.Marshal(uar)
	if err != nil {
		return nil, errs.Wrap(http.StatusInternalServerError, err, "error marshaling request")
	}
	u := c.endpoint.ResolveReference(&url.URL{Path: path.Join("/mgmt/admin", id)})
	req, err := http.NewRequest("PUT", u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrapf(err, "create PUT %s request failed", u)
	}
retry:
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "client PUT %s failed", u)
	}
	if resp.StatusCode >= 400 {
		if !retried && c.retryOnError(resp) {
			retried = true
			goto retry
		}
		return nil, readError(resp.Body)
	}
	var adm = new(mgmt.Admin)
	if err := readJSON(resp.Body, adm); err != nil {
		return nil, errors.Wrapf(err, "error reading %s", u)
	}
	return adm, nil
}

// GetAdmins performs the GET /mgmt/admins request to the CA.
func (c *MgmtClient) GetAdmins() ([]*mgmt.Admin, error) {
	var retried bool
	u := c.endpoint.ResolveReference(&url.URL{Path: "/mgmt/admins"})
retry:
	resp, err := c.client.Get(u.String())
	if err != nil {
		return nil, errors.Wrapf(err, "client GET %s failed", u)
	}
	if resp.StatusCode >= 400 {
		if !retried && c.retryOnError(resp) {
			retried = true
			goto retry
		}
		return nil, readError(resp.Body)
	}
	var admins = new([]*mgmt.Admin)
	if err := readJSON(resp.Body, admins); err != nil {
		return nil, errors.Wrapf(err, "error reading %s", u)
	}
	return *admins, nil
}

// GetProvisioners performs the GET /mgmt/provisioners request to the CA.
func (c *MgmtClient) GetProvisioners() ([]*mgmt.Provisioner, error) {
	var retried bool
	u := c.endpoint.ResolveReference(&url.URL{Path: "/mgmt/provisioners"})
retry:
	resp, err := c.client.Get(u.String())
	if err != nil {
		return nil, errors.Wrapf(err, "client GET %s failed", u)
	}
	if resp.StatusCode >= 400 {
		if !retried && c.retryOnError(resp) {
			retried = true
			goto retry
		}
		return nil, readError(resp.Body)
	}
	var provs = new([]*mgmt.Provisioner)
	if err := readJSON(resp.Body, provs); err != nil {
		return nil, errors.Wrapf(err, "error reading %s", u)
	}
	return *provs, nil
}
