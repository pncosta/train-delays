import 'package:flutter/material.dart';

import '../models/summary.dart';

class LeaderboardEntryCard extends StatelessWidget {
  const LeaderboardEntryCard({super.key, required this.trip});

  final LeaderboardEntry trip;

  @override
  Widget build(BuildContext context) {
    return Card(
      elevation: 2,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      child: Padding(
        padding: const EdgeInsets.all(16.0),
        child: Row(
          // Ensure items are vertically centered in the row
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            Expanded(
              flex: 3,
              // Align keeps the Chip from stretching to fill the whole expanded space
              child: Align(
                alignment: Alignment.centerLeft,
                child: Chip(
                  label: Text(
                    trip.serviceType.toLocalizedString(context).toUpperCase(),
                    // Optionally scale text down if it's too long for the flex space
                    overflow: TextOverflow.ellipsis,
                  ),
                  backgroundColor: getColor(trip.serviceType),
                  labelStyle: TextStyle(
                    color: Colors.blue.shade700,
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ),
            ),
            SizedBox(width: 12),

            // Reduced fixed spacing since Expanded handles distribution
            Expanded(
              flex: 2,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                // Align to the left of its block
                children: [
                  Icon(Icons.train),
                  SizedBox(width: 6),
                  Text(
                    trip.trainNumber,
                    style: TextStyle(fontSize: 22, fontWeight: FontWeight.bold),
                  ),
                ],
              ),
            ),

            SizedBox(width: 12),
            Expanded(
              flex: 4,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Container(
                    padding: EdgeInsets.symmetric(
                      horizontal: 8,
                      vertical: 4,
                    ),
                    decoration: BoxDecoration(
                      color: Colors.red.shade100,
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child:     Text(
                      "${trip.value.round().toString()} min" ,
                      style: TextStyle(
                        color: Colors.red.shade900,
                        fontWeight: FontWeight.bold,
                        fontSize: 12,
                      ),
                    ),
                  ),


                ],
              ),
            ),
            SizedBox(width: 12),

            Expanded(
              flex: 8,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    trip.originStationName,
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w500),
                    overflow: TextOverflow.ellipsis,
                  ),
                  Icon(Icons.arrow_downward, size: 16, color: Colors.grey),
                  Text(
                    trip.destinationStationName,
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w500),
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Color getColor(ServiceType serviceType) {
    switch (serviceType) {
      case ServiceType.Regional:
        return Colors.blue.shade50;
      case ServiceType.Urban:
        return Colors.green.shade50;
      case ServiceType.AlfaPendular:
        return Colors.red.shade50;
      case ServiceType.InterCidades:
        return Colors.yellow.shade50;
      default:
        return Colors.grey.shade50;
    }
  }
}
