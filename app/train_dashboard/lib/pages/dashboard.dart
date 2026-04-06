import 'package:flutter/material.dart';
import 'package:fl_chart/fl_chart.dart';
import '../services/api.dart';
import '../models/summary.dart';
import '../widgets/pie_chart.dart';

class Dashboard extends StatefulWidget {
  const Dashboard({super.key});

  @override
  State<Dashboard> createState() => _DashboardState();
}

class _DashboardState extends State<Dashboard> {
  final api = ApiService();
  late Future<Summary> summaryFuture;

  @override
  void initState() {
    super.initState();
    summaryFuture = api.getSummary(); // Initial load
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<Summary>(
      future: summaryFuture,
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return const Center(child: CircularProgressIndicator());
        } else if (snapshot.hasError) {
          return Center(child: Text('Error: ${snapshot.error}'));
        }

        final data = snapshot.data!;
        return DashboardContent(
          DashboardContentViewModel(
            avgDelay: data.avgDelay,
            totalTrains: data.totalTrains,
            totalCancelled: 3,
            totalDelayed: 100,
          ),
        );
      },
    );
  }
}

class DashboardContentViewModel {
  final double avgDelay;
  final double totalTrains;
  final double totalCancelled;
  final double totalDelayed;

  const DashboardContentViewModel({
    required this.avgDelay,
    required this.totalTrains,
    required this.totalCancelled,
    required this.totalDelayed,
  });
}

class DashboardContent extends StatelessWidget {
  const DashboardContent(this.viewModel, {super.key});

  final DashboardContentViewModel viewModel;

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 16,
      runSpacing: 16,
      // mainAxisSize: MainAxisSize.min,
      alignment: WrapAlignment.spaceAround,
      children: [
        NumberCard(
          icon: Icons.watch_later,
          title: "Atraso médio",
          unit: "Minutos",
          value: viewModel.avgDelay.toString(),
        ),
        NumberCard(
          icon: Icons.numbers,
          title: "Total",
          unit: "Comboios",
          value: viewModel.totalTrains.toString(),
        ),

        ConstrainedBox(
          constraints: BoxConstraints(minWidth: 300),
          child: NumberCard(
            icon: Icons.cancel,
            title: "Cancelados",
            unit: "Comboios",
            value: viewModel.totalCancelled.toString(),
          ),
        ),
        ConstrainedBox(
          constraints: BoxConstraints(minWidth: 300),
          child: NumberCard(
            icon: Icons.watch_off,
            title: "Atrasados",
            unit: "Comboios",
            value: viewModel.totalCancelled.toString(),
          ),
        ),
        // PlotCard(),
        ConstrainedBox(
          constraints: BoxConstraints(minWidth: 300, maxWidth: 500),
          child: Card(child: PieChartTrainStatus(
            onTime: viewModel.totalTrains,
            cancelled: viewModel.totalCancelled,
            delayed: viewModel.totalDelayed,
          )),
        ),
      ],
    );
  }
}

class NumberCard extends StatelessWidget {
  const NumberCard({
    super.key,
    required this.icon,
    required this.value,
    required this.unit,
    required this.title,
  });

  final IconData icon;
  final String value;
  final String unit;
  final String title;

  @override
  Widget build(BuildContext context) {
    return StatsCard(
      icon: icon,
      title: title,
      child: Column(
        children: [
          Text(
            value,
            style: TextStyle(fontWeight: FontWeight.bold, fontSize: 28),
          ),
          Text(unit),
        ],
      ),
    );
  }
}

class StatsCard extends StatelessWidget {
  const StatsCard({
    super.key,
    required this.icon,
    required this.child,
    required this.title,
  });

  final IconData icon;
  final Widget child;
  final String title;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: EdgeInsets.all(8.0),
        child: Column(
          mainAxisSize: MainAxisSize.min,

          children: [
            Icon(icon),
            child,
            Text(title, style: TextStyle(fontWeight: FontWeight.bold)),
          ],
        ),
      ),
    );
  }
}
