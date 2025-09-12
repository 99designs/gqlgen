#!/bin/bash
# this script will update gqlparser to the latest version (because we do it frequently)
# and make a new branch
gh-find-latest() {
  local owner=$1 project=$2
  local release_url=$(curl -Ls -o /dev/null -w '%{url_effective}' "https://github.com/${owner}/${project}/releases/latest")
  export release_tag=$(basename $release_url)
}

# Get Release tag
gh-find-latest vektah gqlparser
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
git commit -s -S -am "Update github.com/vektah/gqlparser/v2@${release_tag}"
go generate ./...
git commit -s -S -am "Re-generate after update"

gh pr create --title "Update gqlparser to $(gh release view -R vektah/gqlparser --json tagName --jq .tagName)" --body "Automated update of gqlparser. See $(gh release view -R vektah/gqlparser --json url --jq .url)" --base "master"
echo "done"


#gh release list --json name,isLatest --jq '.[] | select(.isLatest)|.name'
