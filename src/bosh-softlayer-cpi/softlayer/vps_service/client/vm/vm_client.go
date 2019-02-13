package vm

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"
)

// New creates a new vm API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) *Client {
	return &Client{transport: transport, formats: formats}
}

/*
Client for vm API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

/*
AddVM adds a new vm to the pool
*/
func (a *Client) AddVM(params *AddVMParams) (*AddVMOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewAddVMParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "addVm",
		Method:             "POST",
		PathPattern:        "/vms",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &AddVMReader{formats: a.formats},
		Context:            params.Context,
	})
	if err != nil {
		return nil, err
	}
	return result.(*AddVMOK), nil

}

/*
DeleteVM deletes a vm from pool
*/
func (a *Client) DeleteVM(params *DeleteVMParams) (*DeleteVMNoContent, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDeleteVMParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "deleteVm",
		Method:             "DELETE",
		PathPattern:        "/vms/{cid}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{""},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &DeleteVMReader{formats: a.formats},
		Context:            params.Context,
	})
	if err != nil {
		return nil, err
	}
	return result.(*DeleteVMNoContent), nil

}

/*
FindVmsByDeployment finds vms by deployment name

Multiple deployment values can be provided with comma separated strings
*/
func (a *Client) FindVmsByDeployment(params *FindVmsByDeploymentParams) (*FindVmsByDeploymentOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewFindVmsByDeploymentParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "findVmsByDeployment",
		Method:             "GET",
		PathPattern:        "/vms/findByDeployment",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{""},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &FindVmsByDeploymentReader{formats: a.formats},
		Context:            params.Context,
	})
	if err != nil {
		return nil, err
	}
	return result.(*FindVmsByDeploymentOK), nil

}

/*
FindVmsByFilters finds vms by filters cpu memory private vlan public vlan state
*/
func (a *Client) FindVmsByFilters(params *FindVmsByFiltersParams) (*FindVmsByFiltersOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewFindVmsByFiltersParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "findVmsByFilters",
		Method:             "POST",
		PathPattern:        "/vms/findByFilters",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &FindVmsByFiltersReader{formats: a.formats},
		Context:            params.Context,
	})
	if err != nil {
		return nil, err
	}
	return result.(*FindVmsByFiltersOK), nil

}

/*
FindVmsByStates finds vms by states
*/
func (a *Client) FindVmsByStates(params *FindVmsByStatesParams) (*FindVmsByStatesOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewFindVmsByStatesParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "findVmsByStates",
		Method:             "GET",
		PathPattern:        "/vms/findByState",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{""},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &FindVmsByStatesReader{formats: a.formats},
		Context:            params.Context,
	})
	if err != nil {
		return nil, err
	}
	return result.(*FindVmsByStatesOK), nil

}

/*
GetVMByCid finds vm by ID

Returns a vm when ID < 10.  ID > 10 or nonintegers will simulate API error conditions
*/
func (a *Client) GetVMByCid(params *GetVMByCidParams) (*GetVMByCidOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetVMByCidParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "getVmByCid",
		Method:             "GET",
		PathPattern:        "/vms/{cid}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{""},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &GetVMByCidReader{formats: a.formats},
		Context:            params.Context,
	})
	if err != nil {
		return nil, err
	}
	return result.(*GetVMByCidOK), nil

}

/*
ListVM lists vms of the pool
*/
func (a *Client) ListVM(params *ListVMParams) (*ListVMOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewListVMParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "listVm",
		Method:             "GET",
		PathPattern:        "/vms",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{""},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &ListVMReader{formats: a.formats},
		Context:            params.Context,
	})
	if err != nil {
		return nil, err
	}
	return result.(*ListVMOK), nil

}

/*
OrderVMByFilter orders a free vm by filters cpu memory private vlan public vlan state
*/
func (a *Client) OrderVMByFilter(params *OrderVMByFilterParams) (*OrderVMByFilterOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewOrderVMByFilterParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "orderVmByFilter",
		Method:             "POST",
		PathPattern:        "/vms/order",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &OrderVMByFilterReader{formats: a.formats},
		Context:            params.Context,
	})
	if err != nil {
		return nil, err
	}
	return result.(*OrderVMByFilterOK), nil

}

/*
UpdateVM updates an existing vm
*/
func (a *Client) UpdateVM(params *UpdateVMParams) (*UpdateVMOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewUpdateVMParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "updateVm",
		Method:             "PUT",
		PathPattern:        "/vms",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &UpdateVMReader{formats: a.formats},
		Context:            params.Context,
	})
	if err != nil {
		return nil, err
	}
	return result.(*UpdateVMOK), nil

}

/*
UpdateVMWithState updates a vm in the pool with state
*/
func (a *Client) UpdateVMWithState(params *UpdateVMWithStateParams) (*UpdateVMWithStateOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewUpdateVMWithStateParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "updateVmWithState",
		Method:             "PUT",
		PathPattern:        "/vms/{cid}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &UpdateVMWithStateReader{formats: a.formats},
		Context:            params.Context,
	})
	if err != nil {
		return nil, err
	}
	return result.(*UpdateVMWithStateOK), nil

}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
