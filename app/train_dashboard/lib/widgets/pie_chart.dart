import 'package:fl_chart/fl_chart.dart';

import 'package:flutter/material.dart';

class PieChartTrainStatus extends StatefulWidget {
  const PieChartTrainStatus({
    required this.onTime,
    required this.delayed,
    required this.cancelled,
    super.key,
  });

  final double onTime;
  final double delayed;
  final double cancelled;

  @override
  State<StatefulWidget> createState() => PieChart2State();
}

class PieChart2State extends State<PieChartTrainStatus> {
  int touchedIndex = -1;

  @override
  Widget build(BuildContext context) {
    return AspectRatio(
      aspectRatio: 1.3,
      child: Row(
        children: <Widget>[
          const SizedBox(height: 18),
          Expanded(
            child: AspectRatio(
              aspectRatio: 1,
              child: PieChart(
                PieChartData(
                  pieTouchData: PieTouchData(
                    touchCallback: (FlTouchEvent event, pieTouchResponse) {
                      setState(() {
                        if (!event.isInterestedForInteractions ||
                            pieTouchResponse == null ||
                            pieTouchResponse.touchedSection == null) {
                          touchedIndex = -1;
                          return;
                        }
                        touchedIndex = pieTouchResponse
                            .touchedSection!
                            .touchedSectionIndex;
                      });
                    },
                  ),
                  borderData: FlBorderData(show: false),
                  sectionsSpace: 1,
                  centerSpaceRadius: 5,
                  sections: showingSections(),
                ),
              ),
            ),
          ),
          const Column(
            mainAxisAlignment: MainAxisAlignment.end,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Indicator(color: Colors.greenAccent, text: 'A tempo', isSquare: true),
              SizedBox(height: 4),
              Indicator(color: Colors.yellowAccent, text: 'Atrasado', isSquare: true),
              SizedBox(height: 4),
              Indicator(color: Colors.redAccent, text: 'Cancelado', isSquare: true),
              SizedBox(height: 18),
            ],
          ),
          const SizedBox(width: 28),
        ],
      ),
    );
  }

  List<PieChartSectionData> showingSections() {
    return List.generate(3, (i) {
      final isTouched = i == touchedIndex;
      final fontSize = isTouched ? 25.0 : 16.0;
      final radius = isTouched ? 60.0 : 50.0;
      const shadows = [Shadow(color: Colors.black, blurRadius: 2)];
      return switch (i) {
        0 => PieChartSectionData(
          color: Colors.greenAccent,
          value: widget.onTime,
          title: isTouched ? widget.onTime.toString() : '',
          radius: radius,
          titleStyle: TextStyle(
            fontSize: fontSize,
            fontWeight: FontWeight.bold,

            shadows: shadows,
          ),
        ),
        1 => PieChartSectionData(
          color: Colors.yellowAccent,
          value: widget.delayed,
          title: isTouched ? widget.delayed.toString() : '',
          radius: radius,
          titleStyle: TextStyle(
            fontSize: fontSize,
            fontWeight: FontWeight.bold,

            shadows: shadows,
          ),
        ),
        2 => PieChartSectionData(
          color: Colors.redAccent,
          value: widget.cancelled,
          title: isTouched ?  widget.cancelled.toString() : '',
          radius: radius,
          titleStyle: TextStyle(
            fontSize: fontSize,
            fontWeight: FontWeight.bold,

            shadows: shadows,
          ),
        ),
        _ => throw StateError('Invalid'),
      };
    });
  }
}

class Indicator extends StatelessWidget {
  const Indicator({
    super.key,
    required this.color,
    required this.text,
    required this.isSquare,
    this.size = 16,
    this.textColor,
  });

  final Color color;
  final String text;
  final bool isSquare;
  final double size;
  final Color? textColor;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: <Widget>[
        Container(
          width: size,
          height: size,
          decoration: BoxDecoration(
            shape: isSquare ? BoxShape.rectangle : BoxShape.circle,
            color: color,
          ),
        ),
        const SizedBox(width: 4),
        Text(
          text,
          style: TextStyle(
            fontSize: 16,
            fontWeight: FontWeight.bold,
            color: textColor,
          ),
        ),
      ],
    );
  }
}
