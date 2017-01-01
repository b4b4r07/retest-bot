package travis

import (
	"fmt"
	"sort"
	"time"

	travis "github.com/Ableton/go-travis"
)

// RepoState is a struct representing basic data about Travis CI repos
type RepoState struct {
	Name         string
	State        string
	LastFinished time.Time
	URL          string
}

// RepoStates is a slice of RepoState structs
type RepoStates []RepoState

func (slice RepoStates) Len() int {
	return len(slice)
}

func (slice RepoStates) Less(i, j int) bool {
	return slice[i].LastFinished.Before(slice[j].LastFinished)
}

func (slice RepoStates) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

var (
	client *travis.Client
)

func init() {
	client = travis.NewClient(travis.TRAVIS_API_DEFAULT_URL, "")
}

// AuthenticateWithGitHub provides a way to turn the client into an
// authenticated one via a GitHub token
func AuthenticateWithGitHub(token string) error {
	_, _, err := client.Authentication.UsingGithubToken(token)
	return err
}

// AuthenticateWithTravis provides a way to turn the client into an
// authenticated one via a Travis token
func AuthenticateWithTravis(token string) bool {
	client = travis.NewClient(travis.TRAVIS_API_DEFAULT_URL, token)
	return client.IsAuthenticated()
}

// GetRepoStatesForUser returns a slice of RepoState structs sorted by
// LastFinished, oldest one first
func GetRepoStatesForUser(user string) (RepoStates, error) {
	var repoData RepoStates
	repos, _, err := client.Repositories.Find(&travis.RepositoryListOptions{OwnerName: user})
	if err != nil {
		return repoData, err
	}

	repoData = make(RepoStates, 0, len(repos))

	for _, repo := range repos {
		name := repo.Slug
		state := repo.LastBuildState
		lastFinished, _ := time.Parse(time.RFC3339, repo.LastBuildFinishedAt)
		url := fmt.Sprintf("https://travis-ci.org/%s/builds/%d", name, repo.LastBuildId)

		if state != "" {
			repoData = append(repoData, RepoState{name, state, lastFinished, url})
		}
	}
	sort.Sort(repoData)

	return repoData, nil
}

// GetBuildStateOfRepo returns the RepoState for a specific repo
func GetBuildStateOfRepo(slug string) (RepoState, error) {
	repo, _, err := client.Repositories.GetFromSlug(slug)
	if err != nil {
		return RepoState{}, err
	}
	name := repo.Slug
	state := repo.LastBuildState
	lastFinished, _ := time.Parse(time.RFC3339, repo.LastBuildFinishedAt)
	url := fmt.Sprintf("https://travis-ci.org/%s/builds/%d", name, repo.LastBuildId)

	return RepoState{name, state, lastFinished, url}, nil
}

// RestartLastBuild restarts the last build for a repository
func RestartLastBuild(slug string) (uint, error) {
	repo, _, err := client.Repositories.GetFromSlug(slug)
	if err != nil {
		return 0, err
	}
	_, err = client.Builds.Restart(repo.LastBuildId)

	return repo.LastBuildId, err
}

func RestartBuildFromPR(slug string, n int) error {
	var id uint
	builds, _, _, _, err := client.Builds.ListFromRepository(
		slug,
		&travis.BuildListOptions{EventType: "pull_request"},
	)
	if err != nil {
		return err
	}
	for _, build := range builds {
		if build.PullRequestNumber == uint(n) {
			id = build.Id
			break
		}
	}

	_, err = client.Builds.Restart(id)
	return err
}
