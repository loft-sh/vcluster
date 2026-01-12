package stripe

import (
	"reflect"

	"github.com/stripe/stripe-go/v81/form"
)

//
// Public types
//

// Iter provides a convenient interface
// for iterating over the elements
// returned from paginated list API calls.
// Successive calls to the Next method
// will step through each item in the list,
// fetching pages of items as needed.
// Iterators are not thread-safe, so they should not be consumed
// across multiple goroutines.
type Iter struct {
	cur        interface{}
	err        error
	formValues *form.Values
	list       ListContainer
	listParams ListParams
	meta       *ListMeta
	query      Query
	values     []interface{}
}

// Current returns the most recent item
// visited by a call to Next.
func (it *Iter) Current() interface{} {
	return it.cur
}

// Err returns the error, if any,
// that caused the Iter to stop.
// It must be inspected
// after Next returns false.
func (it *Iter) Err() error {
	return it.err
}

// List returns the current list object which the iterator is currently using.
// List objects will change as new API calls are made to continue pagination.
func (it *Iter) List() ListContainer {
	return it.list
}

// Meta returns the list metadata.
func (it *Iter) Meta() *ListMeta {
	return it.meta
}

// Next advances the Iter to the next item in the list,
// which will then be available
// through the Current method.
// It returns false when the iterator stops
// at the end of the list.
func (it *Iter) Next() bool {
	if len(it.values) == 0 && it.meta.HasMore && !it.listParams.Single {
		// determine if we're moving forward or backwards in paging
		if it.listParams.EndingBefore != nil {
			it.listParams.EndingBefore = String(listItemID(it.cur))
			it.formValues.Set(EndingBefore, *it.listParams.EndingBefore)
		} else {
			it.listParams.StartingAfter = String(listItemID(it.cur))
			it.formValues.Set(StartingAfter, *it.listParams.StartingAfter)
		}
		it.getPage()
	}
	if len(it.values) == 0 {
		return false
	}
	it.cur = it.values[0]
	it.values = it.values[1:]
	return true
}

func (it *Iter) getPage() {
	it.values, it.list, it.err = it.query(it.listParams.GetParams(), it.formValues)
	it.meta = it.list.GetListMeta()

	if it.listParams.EndingBefore != nil {
		// We are moving backward,
		// but items arrive in forward order.
		reverse(it.values)
	}
}

// Query is the function used to get a page listing.
type Query func(*Params, *form.Values) ([]interface{}, ListContainer, error)

//
// Public functions
//

// GetIter returns a new Iter for a given query and its options.
func GetIter(container ListParamsContainer, query Query) *Iter {
	var listParams *ListParams
	formValues := &form.Values{}

	if container != nil {
		reflectValue := reflect.ValueOf(container)

		// See the comment on Call in stripe.go.
		if reflectValue.Kind() == reflect.Ptr && !reflectValue.IsNil() {
			listParams = container.GetListParams()
			form.AppendTo(formValues, container)
		}
	}

	if listParams == nil {
		listParams = &ListParams{}
	}
	iter := &Iter{
		formValues: formValues,
		listParams: *listParams,
		query:      query,
	}

	iter.getPage()

	return iter
}

//
// Private functions
//

func listItemID(x interface{}) string {
	return reflect.ValueOf(x).Elem().FieldByName("ID").String()
}

func reverse(a []interface{}) {
	for i := 0; i < len(a)/2; i++ {
		a[i], a[len(a)-i-1] = a[len(a)-i-1], a[i]
	}
}
