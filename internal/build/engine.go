package build

// Engine will own the full build pipeline: clone, detect, build image,
// stream logs, tag image. Implemented in Day 6.
//
// For now, runtime detection (detect.go) and template selection
// (templates.go) are called directly from internal/worker/pool.go.