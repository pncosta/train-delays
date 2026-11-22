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
	day, serviceType                                        string
	trips, cancelled, onTime, delayed, delaySum, delayCount int
	lateUpTo10, late10To30, lateOver30, maxDelay            int
}

type worstTrip struct {
	day, serviceType string
	rank             int
	trip             trip
	delay            int
}
