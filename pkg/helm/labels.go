package helm

// labels is a map of key value pairs to be included as metadata in a configmap object.
// type labels map[string]string

// func (lbs *labels) init()                { *lbs = labels(make(map[string]string)) }
// func (lbs labels) get(key string) string { return lbs[key] }
// func (lbs labels) set(key, val string)   { lbs[key] = val }

// func (lbs labels) keys() (ls []string) {
// 	for key := range lbs {
// 		ls = append(ls, key)
// 	}
// 	return
// }

// func (lbs labels) match(set labels) bool {
// 	for _, key := range set.keys() {
// 		if lbs.get(key) != set.get(key) {
// 			return false
// 		}
// 	}
// 	return true
// }

// func (lbs labels) toMap() map[string]string { return lbs }

// func (lbs *labels) fromMap(kvs map[string]string) {
// 	for k, v := range kvs {
// 		lbs.set(k, v)
// 	}
// }
