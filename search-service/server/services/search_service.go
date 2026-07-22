package services

import (
	"context"

	searchv1 "github.com/JustUzair/irctc-microservice/gen/go/search/v1"
)

type SearchService struct {
	searchv1.UnimplementedSearchServiceServer
}

func (*SearchService) GetTrain(context.Context, *searchv1.GetTrainRequest) (*searchv1.GetTrainResponse, error) {
	return &searchv1.GetTrainResponse{
		Train: &searchv1.Train{Name: "train"},
	}, nil
}
