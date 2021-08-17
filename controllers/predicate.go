package controllers

import (
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type hasCdnAnnotationPredicate struct{}

var _ predicate.Predicate = &hasCdnAnnotationPredicate{}

func (p hasCdnAnnotationPredicate) Create(event event.CreateEvent) bool {
	return hasCdnAnnotation(event.Object) && hasLoadBalancer(event.Object)
}

func (p hasCdnAnnotationPredicate) Delete(event.DeleteEvent) bool {
	return false
}

func (p hasCdnAnnotationPredicate) Update(event event.UpdateEvent) bool {
	return hasCdnAnnotation(event.ObjectNew) && hasLoadBalancer(event.ObjectNew)
}

func (p hasCdnAnnotationPredicate) Generic(event.GenericEvent) bool {
	return false
}

func hasCdnAnnotation(o client.Object) bool {
	return len(o.GetAnnotations()[cdnIDAnnotation]) > 0
}

func hasLoadBalancer(o client.Object) bool {
	ing, ok := o.(*networkingv1.Ingress)
	if !ok {
		return false
	}

	return len(ing.Status.LoadBalancer.Ingress) > 0
}
