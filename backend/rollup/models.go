package main

// trip mirrors a trips row; NULL columns are nil.
type trip struct {
	id                 string
	trainNumber        *int
	serviceType        *string
	originStation      *string
	destinationStation *string
	scheduledArrival   *string
	actualArrival      *string
	delayMinutes       *int
	isCancelled        *bool
}

type dailyStats struct {
	day         string
	serviceType string

	trips     int
	cancelled int
	onTime    int
	delayed   int

	delaySum   int
	delayCount int
	maxDelay   int

	lateUpTo10 int
	late10To30 int
	lateOver30 int
}

type worstTrip struct {
	day, serviceType string
	rank             int
	trip             trip
	delay            int
}
