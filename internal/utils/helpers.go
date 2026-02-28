package utils

func GetProjectNameByID(projectID string) string {
	switch projectID {
	case "141":
		return "FrontEnd"
	case "140":
		return "BackEnd"
	default:
		return "UndefinedProject"
	}
}

func GetStandBranchByProjectID(projectID, backendStand, frontendStand string) string {
	switch projectID {
	case "140":
		return backendStand
	case "141":
		return frontendStand
	default:
		return "UndefinedProject"
	}
}
