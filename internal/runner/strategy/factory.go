package strategy

func GetRunner(loaderType string) ServerRunner {
	switch loaderType {
	case "forge", "neoforge":
		return &ForgeRunner{}
	case "paper", "vanilla", "fabric":
		return &VanillaRunner{JarName: "server.jar"}
	default:
		return &VanillaRunner{JarName: "server.jar"}
	}
}
