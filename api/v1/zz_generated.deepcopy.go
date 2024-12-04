//go:build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	corev1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MastodonServer) DeepCopyInto(out *MastodonServer) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MastodonServer.
func (in *MastodonServer) DeepCopy() *MastodonServer {
	if in == nil {
		return nil
	}
	out := new(MastodonServer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MastodonServer) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MastodonServerList) DeepCopyInto(out *MastodonServerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MastodonServer, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MastodonServerList.
func (in *MastodonServerList) DeepCopy() *MastodonServerList {
	if in == nil {
		return nil
	}
	out := new(MastodonServerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MastodonServerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MastodonServerMigratingStatus) DeepCopyInto(out *MastodonServerMigratingStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MastodonServerMigratingStatus.
func (in *MastodonServerMigratingStatus) DeepCopy() *MastodonServerMigratingStatus {
	if in == nil {
		return nil
	}
	out := new(MastodonServerMigratingStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MastodonServerSidekiqSpec) DeepCopyInto(out *MastodonServerSidekiqSpec) {
	*out = *in
	if in.EnvFrom != nil {
		in, out := &in.EnvFrom, &out.EnvFrom
		*out = make([]corev1.EnvFromSource, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.PodAnnotations != nil {
		in, out := &in.PodAnnotations, &out.PodAnnotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.PeriodicRestart != nil {
		in, out := &in.PeriodicRestart, &out.PeriodicRestart
		*out = new(PeriodicRestartSpec)
		(*in).DeepCopyInto(*out)
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MastodonServerSidekiqSpec.
func (in *MastodonServerSidekiqSpec) DeepCopy() *MastodonServerSidekiqSpec {
	if in == nil {
		return nil
	}
	out := new(MastodonServerSidekiqSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MastodonServerSpec) DeepCopyInto(out *MastodonServerSpec) {
	*out = *in
	in.Web.DeepCopyInto(&out.Web)
	in.Streaming.DeepCopyInto(&out.Streaming)
	in.Sidekiq.DeepCopyInto(&out.Sidekiq)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MastodonServerSpec.
func (in *MastodonServerSpec) DeepCopy() *MastodonServerSpec {
	if in == nil {
		return nil
	}
	out := new(MastodonServerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MastodonServerStatus) DeepCopyInto(out *MastodonServerStatus) {
	*out = *in
	if in.Migrating != nil {
		in, out := &in.Migrating, &out.Migrating
		*out = new(MastodonServerMigratingStatus)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MastodonServerStatus.
func (in *MastodonServerStatus) DeepCopy() *MastodonServerStatus {
	if in == nil {
		return nil
	}
	out := new(MastodonServerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MastodonServerStreamingSpec) DeepCopyInto(out *MastodonServerStreamingSpec) {
	*out = *in
	if in.EnvFrom != nil {
		in, out := &in.EnvFrom, &out.EnvFrom
		*out = make([]corev1.EnvFromSource, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.PodAnnotations != nil {
		in, out := &in.PodAnnotations, &out.PodAnnotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.PeriodicRestart != nil {
		in, out := &in.PeriodicRestart, &out.PeriodicRestart
		*out = new(PeriodicRestartSpec)
		(*in).DeepCopyInto(*out)
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MastodonServerStreamingSpec.
func (in *MastodonServerStreamingSpec) DeepCopy() *MastodonServerStreamingSpec {
	if in == nil {
		return nil
	}
	out := new(MastodonServerStreamingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MastodonServerWebSpec) DeepCopyInto(out *MastodonServerWebSpec) {
	*out = *in
	if in.EnvFrom != nil {
		in, out := &in.EnvFrom, &out.EnvFrom
		*out = make([]corev1.EnvFromSource, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.PodAnnotations != nil {
		in, out := &in.PodAnnotations, &out.PodAnnotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.PeriodicRestart != nil {
		in, out := &in.PeriodicRestart, &out.PeriodicRestart
		*out = new(PeriodicRestartSpec)
		(*in).DeepCopyInto(*out)
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MastodonServerWebSpec.
func (in *MastodonServerWebSpec) DeepCopy() *MastodonServerWebSpec {
	if in == nil {
		return nil
	}
	out := new(MastodonServerWebSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PeriodicRestartSpec) DeepCopyInto(out *PeriodicRestartSpec) {
	*out = *in
	if in.TimeZone != nil {
		in, out := &in.TimeZone, &out.TimeZone
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PeriodicRestartSpec.
func (in *PeriodicRestartSpec) DeepCopy() *PeriodicRestartSpec {
	if in == nil {
		return nil
	}
	out := new(PeriodicRestartSpec)
	in.DeepCopyInto(out)
	return out
}
