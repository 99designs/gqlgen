#!/bin/bash
# this script will update gqlparser to the latest version (because we do it frequently)
# and make a new branch
gh-find-latest() {
  local owner=$1 project=$2
  local release_url=$(curl -Ls -o /dev/null -w '%{url_effective}' "https://github.com/${owner}/${project}/releases/latest")
  export release_tag=$(basename $release_url)
}

# build_pr_body prints the pull request body for release_tag: a link to the
# release plus its changelog, with gqlparser PR numbers rewritten as absolute
# links so GitHub does not resolve them against gqlgen's own PRs.
# If the changelog cannot be fetched or rewritten, it prints only the link.
build_pr_body() {
  local release_url="${repo_url}/releases/tag/${release_tag}"
  local release_body changelog
  echo "Automated update of gqlparser. See ${release_url}"
  if ! release_body=$(gh release view "${release_tag}" -R "${owner}/${project}" --json body --jq .body); then
    echo "warning: could not fetch release notes for ${release_tag}; PR body has no changelog" >&2
    return 0
  fi
  if ! changelog=$(printf '%s\n' "${release_body}" | go run -C bin/_tools ./changelogabsref "${repo_url}") ||
    [ -z "${changelog}" ]; then
    echo "warning: could not rewrite release notes for ${release_tag}; PR body has no changelog" >&2
    return 0
  fi
  printf '\n%s\n' "${changelog}"
}

owner=vektah
project=gqlparser
repo_url="https://github.com/${owner}/${project}"

# Get Release tag
gh-find-latest "${owner}" "${project}"
echo "Latest Release is ${release_tag}"

export branchName="update_gqlparser_v2_${release_tag}"
echo "${branchName}"
sanitized_branch_name=$(echo ${branchName} | sed -E 's/\s+/\s/g' | sed -E 's/\./_/g')
echo "${sanitized_branch_name}"
git checkout -b "${sanitized_branch_name}"

go get github.com/vektah/gqlparser/v2@${release_tag}
go mod tidy
cd _examples
go get github.com/vektah/gqlparser/v2@${release_tag}
go mod tidy
cd ..
find . -name 'go.mod' -execdir go mod tidy \;
git commit -s -S -am "Update github.com/vektah/gqlparser/v2@${release_tag}"
go generate ./...
git commit -s -S -am "Re-generate after update"

build_pr_body | gh pr create --title "Update gqlparser to ${release_tag}" --body-file - --base "master"
echo "done"


#gh release list --json name,isLatest --jq '.[] | select(.isLatest)|.name'
