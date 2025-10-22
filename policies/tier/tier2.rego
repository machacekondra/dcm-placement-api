package tier2

default valid := false

# Required zones for tier 2
required_zones := data.t2.zones

# Generate failures if zones are defined but not equal to required_zones
failures contains failure if {
	input.zones  # Zones field exists
	some zone in required_zones
	not zone in input.zones
	failure := sprintf("Missing required zone '%s' in input specification", [zone])
}

failures contains failure if {
	input.zones  # Zones field exists
	some zone in input.zones
	not zone in required_zones
	failure := sprintf("Unexpected zone '%s' in input specification", [zone])
}

# Input is valid if zones are not defined OR zones exactly match required_zones
valid if {
	not input.zones  # Zones field does not exist - this is valid
}

valid if {
	input.zones  # Zones field exists
	# All required zones are present
	every zone in required_zones {
		zone in input.zones
	}
	# No extra zones are present
	every zone in input.zones {
		zone in required_zones
	}
}
