//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

type Application struct {
	Deleted bool `json:"deleted"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// The name of the application.
	Name string `json:"name"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
}

// UnmarshalJSON handles deserialization of an Application.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (a *Application) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		a.ID = id
		return nil
	}

	type application Application
	var v application
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*a = Application(v)
	return nil
}
