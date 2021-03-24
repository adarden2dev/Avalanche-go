package indexer

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/utils/json"
)

type service struct {
	Index
}

type FormattedContainer struct {
	ID        ids.ID              `json:"id"`
	Bytes     string              `json:"bytes"`
	Timestamp time.Time           `json:"timestamp"`
	Encoding  formatting.Encoding `json:"encoding"`
	Index     json.Uint64         `json:"index"`
}

func newFormattedContainer(c Container, index uint64, enc formatting.Encoding) (FormattedContainer, error) {
	fc := FormattedContainer{
		Encoding: enc,
		ID:       c.ID,
		Index:    json.Uint64(index),
	}
	bytesStr, err := formatting.Encode(enc, c.Bytes)
	if err != nil {
		return fc, err
	}
	fc.Bytes = bytesStr
	fc.Timestamp = time.Unix(c.Timestamp, 0)
	return fc, nil
}

type GetLastAcceptedArgs struct {
	Encoding formatting.Encoding `json:"encoding"`
}

func (s *service) GetLastAccepted(_ *http.Request, args *GetLastAcceptedArgs, reply *FormattedContainer) error {
	container, err := s.Index.GetLastAccepted()
	if err != nil {
		return err
	}
	index, err := s.Index.GetIndex(container.ID)
	if err != nil {
		return fmt.Errorf("couldn't get index: %s", err)
	}
	*reply, err = newFormattedContainer(container, index, args.Encoding)
	return err
}

type GetContainer struct {
	Index    json.Uint64         `json:"index"`
	Encoding formatting.Encoding `json:"encoding"`
}

func (s *service) GetContainerByIndex(_ *http.Request, args *GetContainer, reply *FormattedContainer) error {
	container, err := s.Index.GetContainerByIndex(uint64(args.Index))
	if err != nil {
		return err
	}
	index, err := s.Index.GetIndex(container.ID)
	if err != nil {
		return fmt.Errorf("couldn't get index: %s", err)
	}
	*reply, err = newFormattedContainer(container, index, args.Encoding)
	return err
}

type GetContainerRange struct {
	StartIndex json.Uint64         `json:"startIndex"`
	NumToFetch json.Uint64         `json:"numToFetch"`
	Encoding   formatting.Encoding `json:"encoding"`
}

// GetContainerRange returns the transactions at index [startIndex], [startIndex+1], ... , [startIndex+n-1]
// If [n] == 0, returns an empty response (i.e. null).
// If [startIndex] > the last accepted index, returns an error (unless the above apply.)
// If [n] > [MaxFetchedByRange], returns an error.
// If we run out of transactions, returns the ones fetched before running out.
func (s *service) GetContainerRange(r *http.Request, args *GetContainerRange, reply *[]FormattedContainer) error {
	containers, err := s.Index.GetContainerRange(uint64(args.StartIndex), uint64(args.NumToFetch))
	if err != nil {
		return err
	}

	formattedContainers := make([]FormattedContainer, len(containers))
	for i, container := range containers {
		index, err := s.Index.GetIndex(container.ID)
		if err != nil {
			return fmt.Errorf("couldn't get index: %s", err)
		}
		formattedContainers[i], err = newFormattedContainer(container, index, args.Encoding)
		if err != nil {
			return err
		}
	}

	*reply = formattedContainers
	return nil
}

type GetIndexArgs struct {
	ContainerID ids.ID              `json:"containerID"`
	Encoding    formatting.Encoding `json:"encoding"`
}

type GetIndexResponse struct {
	Index json.Uint64 `json:"index"`
}

func (s *service) GetIndex(r *http.Request, args *GetIndexArgs, reply *GetIndexResponse) error {
	index, err := s.Index.GetIndex(args.ContainerID)
	reply.Index = json.Uint64(index)
	return err
}

func (s *service) IsAccepted(r *http.Request, args *GetIndexArgs, reply *bool) error {
	_, err := s.Index.GetIndex(args.ContainerID)
	*reply = err == nil
	return nil
}

func (s *service) GetContainerByID(r *http.Request, args *GetIndexArgs, reply *FormattedContainer) error {
	container, err := s.Index.GetContainerByID(args.ContainerID)
	if err != nil {
		return err
	}
	index, err := s.Index.GetIndex(container.ID)
	if err != nil {
		return fmt.Errorf("couldn't get index: %s", err)
	}
	*reply, err = newFormattedContainer(container, index, args.Encoding)
	return err
}
