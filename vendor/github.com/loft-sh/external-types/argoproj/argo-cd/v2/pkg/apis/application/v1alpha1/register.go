package v1alpha1

func init() {
	SchemeBuilder.Register(&AppProject{}, &AppProjectList{})
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
