package route

import "github.com/wyw14/cry-111/internal/model"

func canTransition(from, to model.RoutePhase) bool {
	if from == to {
		return true
	}
	switch from {
	case model.RouteRequested:
		return to == model.RouteProving || to == model.RouteCancelled
	case model.RouteProving:
		return to == model.RouteLocked || to == model.RouteCancelled
	case model.RouteLocked:
		return to == model.RouteSignalled || to == model.RouteReleasing || to == model.RouteCancelled
	case model.RouteSignalled:
		return to == model.RouteOccupied || to == model.RouteReleasing
	case model.RouteOccupied:
		return to == model.RouteReleasing
	case model.RouteReleasing:
		return to == model.RouteNormal || to == model.RouteCancelled
	default:
		return false
	}
}

func safeRecoveryPhase(route model.Route) model.RoutePhase {
	if !route.Active() {
		return route.Phase
	}
	return model.RouteProving
}

func phaseRank(phase model.RoutePhase) int {
	switch phase {
	case model.RouteRequested:
		return 1
	case model.RouteProving:
		return 2
	case model.RouteLocked:
		return 3
	case model.RouteSignalled:
		return 4
	case model.RouteOccupied:
		return 5
	case model.RouteReleasing:
		return 6
	case model.RouteNormal, model.RouteCancelled:
		return 7
	default:
		return 0
	}
}

func IsAtLeast(route model.Route, phase model.RoutePhase) bool {
	return phaseRank(route.Phase) >= phaseRank(phase)
}
