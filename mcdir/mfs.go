package main

import (
	pb "github.com/mediachain/concat/proto"
)

type ManifestStoreImpl struct {
}

func NewManifestStore() ManifestStore {
	return &ManifestStoreImpl{}
}

func (mfs *ManifestStoreImpl) Put([]*pb.Manifest) {

}

func (mfs *ManifestStoreImpl) Lookup(entity string) []*pb.Manifest {
	return nil
}
