package cluster

import (
	"context"
	"fmt"
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	kkarlonapp "github.com/jayanth-tjvrr/karlon/pkg/app"
)

//------------------------------------------------------------------------------

func SetAppProfiles(
	appIf argoapp.ApplicationServiceClient,
	name string,
	commaSeparatedAppProfiles string,
) error {
	app, err := appIf.Get(context.Background(),
		&argoapp.ApplicationQuery{
			Name: &name,
		})
	if err != nil {
		return fmt.Errorf("failed to get argocd application: %s", err)
	}
	if app.Labels["kkarlon-type"] != "cluster-app" {
		return fmt.Errorf("application resource is not an Arlon cluster")
	}
	app.Annotations[kkarlonapp.ProfilesAnnotationKey] = commaSeparatedAppProfiles
	_, err = appIf.Update(context.Background(), &argoapp.ApplicationUpdateRequest{
		Application: app,
	})
	if err != nil {
		return fmt.Errorf("failed to update argocd application: %s", err)
	}
	return nil
}
