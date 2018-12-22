workflow "PR" {
  on = "push"
  resolves = ["Build"]
}

action "Lint" {
  uses = "./ci/gotests/"
  runs = "make"
  args = "lint"
}

action "Gofmt" {
  uses = "./ci/gotests/"
  runs = "make"
  args = "fmt"
}

action "Build" {
  needs = ["Lint", "Gofmt"]
  uses = "./ci/gotests/"
  runs = "make"
  args = "build"
}
