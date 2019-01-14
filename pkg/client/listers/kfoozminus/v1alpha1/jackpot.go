/*
Custom License for kfoozminus Projects
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/kfoozminus/booklist-controller/pkg/apis/kfoozminus/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// JackpotLister helps list Jackpots.
type JackpotLister interface {
	// List lists all Jackpots in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.Jackpot, err error)
	// Jackpots returns an object that can list and get Jackpots.
	Jackpots(namespace string) JackpotNamespaceLister
	JackpotListerExpansion
}

// jackpotLister implements the JackpotLister interface.
type jackpotLister struct {
	indexer cache.Indexer
}

// NewJackpotLister returns a new JackpotLister.
func NewJackpotLister(indexer cache.Indexer) JackpotLister {
	return &jackpotLister{indexer: indexer}
}

// List lists all Jackpots in the indexer.
func (s *jackpotLister) List(selector labels.Selector) (ret []*v1alpha1.Jackpot, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Jackpot))
	})
	return ret, err
}

// Jackpots returns an object that can list and get Jackpots.
func (s *jackpotLister) Jackpots(namespace string) JackpotNamespaceLister {
	return jackpotNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// JackpotNamespaceLister helps list and get Jackpots.
type JackpotNamespaceLister interface {
	// List lists all Jackpots in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.Jackpot, err error)
	// Get retrieves the Jackpot from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.Jackpot, error)
	JackpotNamespaceListerExpansion
}

// jackpotNamespaceLister implements the JackpotNamespaceLister
// interface.
type jackpotNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Jackpots in the indexer for a given namespace.
func (s jackpotNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.Jackpot, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Jackpot))
	})
	return ret, err
}

// Get retrieves the Jackpot from the indexer for a given namespace and name.
func (s jackpotNamespaceLister) Get(name string) (*v1alpha1.Jackpot, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("jackpot"), name)
	}
	return obj.(*v1alpha1.Jackpot), nil
}
