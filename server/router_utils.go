package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// RouterMissingParamError missing parameter in URL
type RouterMissingParamError struct {
	paramName string
}

func (e *RouterMissingParamError) Error() string { return fmt.Sprintf("missing param %v", e.paramName) }

// RouterParsingError parsing error, for example string when we expected integer
type RouterParsingError struct {
	paramName  string
	paramValue interface{}
	errString  string
}

func (e *RouterParsingError) Error() string {
	return fmt.Sprintf(
		"Error during parsing param %v with value %v. Error: %v",
		e.paramName, e.paramValue, e.errString,
	)
}

// GetRouterParam retrieves parameter from URL like `/organization/{org_id}`
func GetRouterParam(request *http.Request, paramName string) (string, error) {
	value, found := mux.Vars(request)[paramName]
	if !found {
		return "", &RouterMissingParamError{paramName: paramName}
	}

	return value, nil
}

// GetRouterIntParam retrieves parameter from URL like `/organization/{org_id}`
// and check it for being valid integer, otherwise returns error
func GetRouterIntParam(request *http.Request, paramName string) (int64, error) {
	value, err := GetRouterParam(request, paramName)
	if err != nil {
		return 0, err
	}
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, &RouterParsingError{
			paramName: paramName, paramValue: value, errString: "integer expected",
		}
	}
	return intValue, nil
}

// GetRouterPositiveIntParam retrieves parameter from URL like `/organization/{org_id}`
// and check it for being valid and positive integer, otherwise returns error
func GetRouterPositiveIntParam(request *http.Request, paramName string) (int64, error) {
	value, err := GetRouterIntParam(request, paramName)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, &RouterParsingError{
			paramName: paramName, paramValue: value, errString: "positive integer expected",
		}
	}
	return value, nil
}

func (server HTTPServer) readClusterName(writer http.ResponseWriter, request *http.Request) (types.ClusterName, error) {
	clusterName, found := mux.Vars(request)["cluster"]
	if !found {
		// query parameter 'cluster' can't be found in request, which might be caused by issue in Gorilla mux
		// (not on client side)
		const message = "Cluster name is not provided"
		log.Println(message)
		responses.SendInternalServerError(writer, message)
		return types.ClusterName(""), errors.New(message)
	}

	if _, err := uuid.Parse(clusterName); err != nil {
		const message = "Cluster name format is invalid"
		log.Println(message)
		responses.SendInternalServerError(writer, message)
		return types.ClusterName(""), errors.New(message)
	}
	return types.ClusterName(clusterName), nil
}

func (server HTTPServer) readOrganizationID(writer http.ResponseWriter, request *http.Request) (types.OrgID, error) {
	organizationID, err := GetRouterPositiveIntParam(request, "organization")
	if err != nil {
		message := fmt.Sprintf("Error getting organization ID from request %v", err.Error())
		log.Println(message)
		if _, ok := err.(*RouterParsingError); ok {
			responses.Send(http.StatusBadRequest, writer, err.Error())
		} else {
			responses.Send(http.StatusInternalServerError, writer, err.Error())
		}
		return 0, errors.New(message)
	}

	return types.OrgID(int(organizationID)), nil
}
