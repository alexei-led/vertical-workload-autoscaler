//go:build !ignore_autogenerated

/*
Copyright 2024 Alexei Ledenev.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Duration) DeepCopyInto(out *Duration) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Duration.
func (in *Duration) DeepCopy() *Duration {
	if in == nil {
		return nil
	}
	out := new(Duration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceReference) DeepCopyInto(out *ResourceReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceReference.
func (in *ResourceReference) DeepCopy() *ResourceReference {
	if in == nil {
		return nil
	}
	out := new(ResourceReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceRequests) DeepCopyInto(out *ResourceRequests) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceRequests.
func (in *ResourceRequests) DeepCopy() *ResourceRequests {
	if in == nil {
		return nil
	}
	out := new(ResourceRequests)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetSpec) DeepCopyInto(out *TargetSpec) {
	*out = *in
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		*out = new(v1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	if in.ResourceReference != nil {
		in, out := &in.ResourceReference, &out.ResourceReference
		*out = new(ResourceReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetSpec.
func (in *TargetSpec) DeepCopy() *TargetSpec {
	if in == nil {
		return nil
	}
	out := new(TargetSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetedResource) DeepCopyInto(out *TargetedResource) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetedResource.
func (in *TargetedResource) DeepCopy() *TargetedResource {
	if in == nil {
		return nil
	}
	out := new(TargetedResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UpdateWindow) DeepCopyInto(out *UpdateWindow) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UpdateWindow.
func (in *UpdateWindow) DeepCopy() *UpdateWindow {
	if in == nil {
		return nil
	}
	out := new(UpdateWindow)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VPAReference) DeepCopyInto(out *VPAReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VPAReference.
func (in *VPAReference) DeepCopy() *VPAReference {
	if in == nil {
		return nil
	}
	out := new(VPAReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkloadAutoscaler) DeepCopyInto(out *WorkloadAutoscaler) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkloadAutoscaler.
func (in *WorkloadAutoscaler) DeepCopy() *WorkloadAutoscaler {
	if in == nil {
		return nil
	}
	out := new(WorkloadAutoscaler)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WorkloadAutoscaler) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkloadAutoscalerList) DeepCopyInto(out *WorkloadAutoscalerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]WorkloadAutoscaler, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkloadAutoscalerList.
func (in *WorkloadAutoscalerList) DeepCopy() *WorkloadAutoscalerList {
	if in == nil {
		return nil
	}
	out := new(WorkloadAutoscalerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WorkloadAutoscalerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkloadAutoscalerSpec) DeepCopyInto(out *WorkloadAutoscalerSpec) {
	*out = *in
	in.Target.DeepCopyInto(&out.Target)
	out.VPAReference = in.VPAReference
	out.UpdateFrequency = in.UpdateFrequency
	if in.AllowedUpdateWindows != nil {
		in, out := &in.AllowedUpdateWindows, &out.AllowedUpdateWindows
		*out = make([]UpdateWindow, len(*in))
		copy(*out, *in)
	}
	out.StepSize = in.StepSize
	out.GracePeriod = in.GracePeriod
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkloadAutoscalerSpec.
func (in *WorkloadAutoscalerSpec) DeepCopy() *WorkloadAutoscalerSpec {
	if in == nil {
		return nil
	}
	out := new(WorkloadAutoscalerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkloadAutoscalerStatus) DeepCopyInto(out *WorkloadAutoscalerStatus) {
	*out = *in
	out.TargetedResource = in.TargetedResource
	in.LastUpdated.DeepCopyInto(&out.LastUpdated)
	out.CurrentRequests = in.CurrentRequests
	out.RecommendedRequests = in.RecommendedRequests
	out.StepSize = in.StepSize
	if in.Errors != nil {
		in, out := &in.Errors, &out.Errors
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkloadAutoscalerStatus.
func (in *WorkloadAutoscalerStatus) DeepCopy() *WorkloadAutoscalerStatus {
	if in == nil {
		return nil
	}
	out := new(WorkloadAutoscalerStatus)
	in.DeepCopyInto(out)
	return out
}
