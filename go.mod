module github.com/123amramr123amr-svg/manus-ai-web

go 1.22

require github.com/tigusigalpa/manus-ai-go/v2 v2.2.2

// The Manus AI Go SDK is vendored under third_party so the project builds
// fully offline. Upstream: https://github.com/tigusigalpa/manus-ai-go (MIT).
replace github.com/tigusigalpa/manus-ai-go/v2 => ./third_party/manus-ai-go
