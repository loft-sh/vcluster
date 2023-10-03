package v1

import (
	admintypes "github.com/loft-sh/external-types/loft-sh/admin-services/pkg/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Announcement holds the announcement information
// +k8s:openapi-gen=true
// +resource:path=announcements,rest=AnnouncementREST
type Announcement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AnnouncementSpec   `json:"spec,omitempty"`
	Status AnnouncementStatus `json:"status,omitempty"`
}

type AnnouncementSpec struct {
}

type AnnouncementStatus struct {
	// Announcement is the html announcement that should be displayed in the frontend
	// +optional
	Announcement string `json:"announcement,omitempty"`

	// InstanceTokenAuth is an instance auth token signed for the user that is used by the frontend
	// +optional
	InstanceTokenAuth *admintypes.InstanceTokenAuth `json:"analyticsToken,omitempty"`
}
