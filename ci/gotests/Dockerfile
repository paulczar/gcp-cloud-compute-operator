FROM go:1.11

LABEL "com.github.actions.name"="Go Tests"
LABEL "com.github.actions.description"="Run Go tests"
LABEL "com.github.actions.color"="purple"

LABEL "repository"="http://github.com/paulczar/gcp-cloud-compute-operator"

RUN \
  go get honnef.co/go/tools/cmd/staticcheck && \
	go get golang.org/x/lint/golint && \
	go get github.com/operator-framework/operator-sdk
