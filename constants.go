package bundleinstall

const (
	// GemsDependency is the name of the dependency provided by the buildpack in
	// its build plan entries.
	GemsDependency = "gems"

	// BundlerDependency is the name of a dependency required by the buildpack in
	// its build plan entries.
	BundlerDependency = "bundler"

	// MRIDependency is the name of a dependency required by the buildpack in its
	// build plan entries.
	MRIDependency = "mri"

	// LayerNameBuildGems is the name of the layer that is used to store gems
	// that are available during the build phase.
	LayerNameBuildGems = "build-gems"

	// LayerNameLaunchGems is the name of the layer that is used to store gems
	// that are available during the launch phase.
	LayerNameLaunchGems = "launch-gems"
)
