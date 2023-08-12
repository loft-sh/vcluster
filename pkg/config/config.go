package config

import (
	"regexp"
)

const Version = "v1beta1"

type Config struct {
	// Version is the config version
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Exports syncs a resource from the virtual cluster to the host
	Exports []*Export `yaml:"export,omitempty" json:"export,omitempty"`

	// Imports syncs a resource from the host cluster to virtual cluster
	Imports []*Import `yaml:"import,omitempty" json:"import,omitempty"`

	// Hooks are hooks that can be used to inject custom patches before syncing
	Hooks *Hooks `yaml:"hooks,omitempty" json:"hooks,omitempty"`
}

type Hooks struct {
	// HostToVirtual is a hook that is executed before syncing from the host to the virtual cluster
	HostToVirtual []*Hook `yaml:"hostToVirtual,omitempty" json:"hostToVirtual,omitempty"`

	// VirtualToHost is a hook that is executed before syncing from the virtual to the host cluster
	VirtualToHost []*Hook `yaml:"virtualToHost,omitempty" json:"virtualToHost,omitempty"`
}

type Hook struct {
	TypeInformation

	// Verbs are the verbs that the hook should mutate
	Verbs []string `yaml:"verbs,omitempty" json:"verbs,omitempty"`

	// Patches are the patches to apply on the object to be synced
	Patches []*Patch `yaml:"patches,omitempty" json:"patches,omitempty"`
}

type Import struct {
	SyncBase `yaml:",inline" json:",inline"`
}

type SyncBase struct {
	TypeInformation `yaml:",inline" json:",inline"`

	Optional bool `yaml:"optional,omitempty" json:"optional,omitempty"`

	// ReplaceOnConflict determines if the controller should try to recreate the object
	// if there is a problem applying
	ReplaceOnConflict bool `yaml:"replaceOnConflict,omitempty" json:"replaceOnConflict,omitempty"`

	// Patches are the patches to apply on the virtual cluster objects
	// when syncing them from the host cluster
	Patches []*Patch `yaml:"patches,omitempty" json:"patches,omitempty"`

	// ReversePatches are the patches to apply to host cluster objects
	// after it has been synced to the virtual cluster
	ReversePatches []*Patch `yaml:"reversePatches,omitempty" json:"reversePatches,omitempty"`
}

type Export struct {
	SyncBase `yaml:",inline" json:",inline"`

	// Selector is a label selector to select the synced objects in the virtual cluster.
	// If empty, all objects will be synced.
	Selector *Selector `yaml:"selector,omitempty" json:"selector,omitempty"`
}

type TypeInformation struct {
	// APIVersion of the object to sync
	APIVersion string `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`

	// Kind of the object to sync
	Kind string `yaml:"kind,omitempty" json:"kind,omitempty"`
}

type Selector struct {
	// LabelSelector are the labels to select the object from
	LabelSelector map[string]string `yaml:"labelSelector,omitempty" json:"labelSelector,omitempty"`
}

type Patch struct {
	// Operation is the type of the patch
	Operation PatchType `yaml:"op,omitempty" json:"op,omitempty"`

	// FromPath is the path from the other object
	FromPath string `yaml:"fromPath,omitempty" json:"fromPath,omitempty"`

	// Path is the path of the patch
	Path string `yaml:"path,omitempty" json:"path,omitempty"`

	// NamePath is the path to the name of a child resource within Path
	NamePath string `yaml:"namePath,omitempty" json:"namePath,omitempty"`

	// NamespacePath is path to the namespace of a child resource within Path
	NamespacePath string `yaml:"namespacePath,omitempty" json:"namespacePath,omitempty"`

	// Value is the new value to be set to the path
	Value interface{} `yaml:"value,omitempty" json:"value,omitempty"`

	// Regex - is regular expresion used to identify the Name,
	// and optionally Namespace, parts of the field value that
	// will be replaced with the rewritten Name and/or Namespace
	Regex       string         `yaml:"regex,omitempty" json:"regex,omitempty"`
	ParsedRegex *regexp.Regexp `yaml:"-" json:"-"`

	// Conditions are conditions that must be true for
	// the patch to get executed
	Conditions []*PatchCondition `yaml:"conditions,omitempty" json:"conditions,omitempty"`

	// Ignore determines if the path should be ignored if handled as a reverse patch
	Ignore *bool `yaml:"ignore,omitempty" json:"ignore,omitempty"`

	// Sync defines if a specialized syncer should be initialized using values
	// from the rewriteName operation as Secret/Configmap names to be synced
	Sync *PatchSync `yaml:"sync,omitempty" json:"sync,omitempty"`
}

type PatchType string

const (
	PatchTypeRewriteName                     = "rewriteName"
	PatchTypeRewriteLabelKey                 = "rewriteLabelKey"
	PatchTypeRewriteLabelSelector            = "rewriteLabelSelector"
	PatchTypeRewriteLabelExpressionsSelector = "rewriteLabelExpressionsSelector"

	PatchTypeCopyFromObject = "copyFromObject"
	PatchTypeAdd            = "add"
	PatchTypeReplace        = "replace"
	PatchTypeRemove         = "remove"
)

type PatchCondition struct {
	// Path is the path within the object to select
	Path string `yaml:"path,omitempty" json:"path,omitempty"`

	// SubPath is the path below the selected object to select
	SubPath string `yaml:"subPath,omitempty" json:"subPath,omitempty"`

	// Equal is the value the path should be equal to
	Equal interface{} `yaml:"equal,omitempty" json:"equal,omitempty"`

	// NotEqual is the value the path should not be equal to
	NotEqual interface{} `yaml:"notEqual,omitempty" json:"notEqual,omitempty"`

	// Empty means that the path value should be empty or unset
	Empty *bool `yaml:"empty,omitempty" json:"empty,omitempty"`
}

type PatchSync struct {
	Secret    *bool `yaml:"secret,omitempty" json:"secret,omitempty"`
	ConfigMap *bool `yaml:"configmap,omitempty" json:"configmap,omitempty"`
}
