package profile

import (
	"embed"
	"fmt"
	kkarlonv1 "github.com/jayanth-tjvrr/karlon/pkg/api/v1"
	"github.com/jayanth-tjvrr/karlon/pkg/argocd"
	"github.com/jayanth-tjvrr/karlon/pkg/bundle"
	"github.com/jayanth-tjvrr/karlon/pkg/common"
	"github.com/jayanth-tjvrr/karlon/pkg/gitutils"
	"github.com/jayanth-tjvrr/karlon/pkg/log"
	gogit "github.com/go-git/go-git/v5"
	"path"
)

//go:embed manifests/*
var content embed.FS

func createInGit(
	creds *argocd.RepoCreds,
	profile *kkarlonv1.Profile,
	kkarlonNs string,
	bundles []bundle.Bundle,
) error {
	log := log.GetLogger()
	repoUrl := profile.Spec.RepoUrl
	repoPath := profile.Spec.RepoPath
	repoRevision := profile.Spec.RepoRevision
	repo, tmpDir, auth, err := argocd.CloneRepo(creds, repoUrl, repoRevision)
	if err != nil {
		return fmt.Errorf("failed to clone repo: %s", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get repo worktree: %s", err)
	}
	// remove old data if directory exists, we'll regenerate everything
	fileInfo, err := wt.Filesystem.Lstat(repoPath)
	if err == nil {
		if !fileInfo.IsDir() {
			return fmt.Errorf("unexpected file type for %s", repoPath)
		}
		_, err = wt.Remove(repoPath)
		if err != nil {
			return fmt.Errorf("failed to recursively delete cluster directory: %s", err)
		}
	}
	mgmtPath := path.Join(repoPath, "mgmt")
	err = gitutils.CopyManifests(wt, content, ".", mgmtPath)
	if err != nil {
		return fmt.Errorf("failed to copy embedded content: %s", err)
	}
	workloadPath := path.Join(repoPath, "workload")
	om := MakeOverridesMap(profile)
	err = gitutils.ProcessBundles(wt, "{{ .Values.clusterName }}", repoUrl,
		mgmtPath, workloadPath, bundles, om)
	if err != nil {
		return fmt.Errorf("failed to process bundles: %s", err)
	}
	changed, err := gitutils.CommitChanges(tmpDir, wt, "manage kkarlon profile "+repoPath)
	if err != nil {
		return fmt.Errorf("failed to commit changes: %s", err)
	}
	if !changed {
		log.Info("no changed files, skipping commit & push")
		return nil
	}
	err = repo.Push(&gogit.PushOptions{
		RemoteName: gogit.DefaultRemoteName,
		Auth:       auth,
		Progress:   nil,
		CABundle:   nil,
	})
	if err != nil {
		return fmt.Errorf("failed to push to remote repository: %s", err)
	}
	log.V(1).Info("successfully pushed working tree", "tmpDir", tmpDir)
	return nil
}

func MakeOverridesMap(profile *kkarlonv1.Profile) (om common.KVPairMap) {
	if len(profile.Spec.Overrides) == 0 {
		return
	}
	om = make(common.KVPairMap)
	for _, item := range profile.Spec.Overrides {
		om[item.Bundle] = append(om[item.Bundle], common.KVPair{
			Key:   item.Key,
			Value: item.Value,
		})
	}
	return
}
