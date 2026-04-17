import 'package:flutter/material.dart';
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

        final summary = snapshot.data!;
        return DashboardContent(summary: summary);
      },
    );
  }
}

class DashboardContent extends StatefulWidget {
  final Summary summary;

  const DashboardContent({required this.summary, super.key});

  @override
  State<DashboardContent> createState() => _DashboardContentState();
}

class _DashboardContentState extends State<DashboardContent> {
  late ServiceTypeStats currentStats;

  @override
  void initState() {
    super.initState();
    currentStats = widget.summary.totalSystem;
  }

  void updateStats(ServiceTypes serviceType) {
    setState(() {
      currentStats =
          widget.summary.breakdown[serviceType] ?? widget.summary.totalSystem;
    });
  }

  @override
  Widget build(BuildContext context) {
    const minSize = 250.0;
    final serviceStats = widget.summary.serviceStats;

    return SingleChildScrollView(
      child: Padding(
        padding: const EdgeInsets.all(16.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            MultipleChoice(onChanged: updateStats),
            // Summary Cards
            Text(
              "Estatisticas da última semana",
              style: Theme.of(context).textTheme.displaySmall,
            ),
            Padding(padding: EdgeInsetsGeometry.symmetric(vertical: 16.0)),
            Wrap(
              spacing: 16,
              runSpacing: 16,
              alignment: WrapAlignment.spaceAround,
              children: [
                ConstrainedBox(
                  constraints: const BoxConstraints(minWidth: minSize),
                  child: NumberCard(
                    icon: Icons.train,
                    title: "Total",
                    unit: "Comboios",
                    value: currentStats.totalTrips.toString(),
                  ),
                ),
                ConstrainedBox(
                  constraints: const BoxConstraints(minWidth: minSize),
                  child: NumberCard(
                    bgColor: Colors.greenAccent,
                    icon: Icons.watch,
                    title: "A horas",
                    unit: "Comboios",
                    value: currentStats.onTimeCount.toString(),
                  ),
                ),
                ConstrainedBox(
                  constraints: const BoxConstraints(minWidth: minSize),
                  child: NumberCard(
                    bgColor: Colors.yellowAccent,
                    icon: Icons.watch_off,
                    title: "Atrasados",
                    unit: "Comboios",
                    value: currentStats.delayedCount.toString(),
                  ),
                ),
                ConstrainedBox(
                  constraints: const BoxConstraints(minWidth: minSize),
                  child: NumberCard(
                    bgColor: Colors.redAccent,
                    icon: Icons.cancel,
                    title: "Cancelados",
                    unit: "Comboios",
                    value: currentStats.cancelledCount.toString(),
                  ),
                ),
                ConstrainedBox(
                  constraints: const BoxConstraints(minWidth: minSize),
                  child: NumberCard(
                    icon: Icons.watch_later,
                    title: "Atraso médio",
                    unit: "Minutos",
                    value: currentStats.avgDelay.round().toString(),
                  ),
                ),

                // Pie Chart
                ConstrainedBox(
                  constraints: const BoxConstraints(
                    minWidth: 300,
                    maxWidth: 300,
                  ),
                  child: Card(
                    child: PieChartTrainStatus(
                      onTime: currentStats.onTimeCount.toDouble(),
                      cancelled: currentStats.cancelledCount.toDouble(),
                      delayed: currentStats.delayedCount.toDouble(),
                    ),
                  ),
                ),
              ],
            ),

            const SizedBox(height: 24),

            // Service Type Breakdown
            const Text(
              'Breakdown by Service Type',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 12),
            ServiceTypeBreakdownTable(stats: serviceStats),
          ],
        ),
      ),
    );
  }
}

class ServiceTypeBreakdownTable extends StatelessWidget {
  final List<ServiceTypeStats> stats;

  const ServiceTypeBreakdownTable({required this.stats, super.key});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: DataTable(
          columns: const [
            DataColumn(label: Text('Service Type')),
            DataColumn(label: Text('Total')),
            DataColumn(label: Text('Avg Delay')),
            DataColumn(label: Text('On Time')),
            DataColumn(label: Text('Delayed')),
            DataColumn(label: Text('Cancelled')),
          ],
          rows: stats.map((stat) {
            return DataRow(
              cells: [
                DataCell(Text(stat.serviceType.toLocalizedString(context))),
                DataCell(Text(stat.totalTrips.toString())),
                DataCell(Text(stat.avgDelay.toStringAsFixed(2))),
                DataCell(
                  Text(
                    '${stat.onTimeCount} (${stat.onTimePercentage.toStringAsFixed(1)}%)',
                  ),
                ),
                DataCell(
                  Text(
                    '${stat.delayedCount} (${stat.delayedPercentage.toStringAsFixed(1)}%)',
                  ),
                ),
                DataCell(Text(stat.cancelledCount.toString())),
              ],
            );
          }).toList(),
        ),
      ),
    );
  }
}

class MultipleChoice extends StatefulWidget {
  const MultipleChoice({super.key, this.onChanged});

  final ValueChanged<ServiceTypes>? onChanged;

  @override
  State<MultipleChoice> createState() => _MultipleChoiceState();
}

class _MultipleChoiceState extends State<MultipleChoice> {
  ServiceTypes selectedType = ServiceTypes.Total;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Text('Serviços', style: Theme.of(context).textTheme.bodyMedium),
        const SizedBox(height: 5.0),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          alignment: WrapAlignment.spaceAround,
          children: ServiceTypes.values.map((ServiceTypes serviceType) {
            return FilterChip(
              label: Text(serviceType.toLocalizedString(context)),
              selected: selectedType == serviceType,
              onSelected: (bool selected) {
                setState(() {
                  if (selected) {
                    selectedType = serviceType;
                  }
                  widget.onChanged?.call(serviceType);
                });
              },
            );
          }).toList(),
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
    this.bgColor = Colors.white,
  });

  final IconData icon;
  final String value;
  final String unit;
  final String title;
  final Color bgColor;

  @override
  Widget build(BuildContext context) {
    return StatsCard(
      bgColor: bgColor,
      icon: icon,
      title: title,
      child: Column(
        children: [
          Text(
            value,
            style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 32),
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
    this.bgColor = Colors.white,
  });

  final IconData icon;
  final Widget child;
  final String title;
  final Color bgColor;

  @override
  Widget build(BuildContext context) {
    return Card(
      color: bgColor,
      child: Padding(
        padding: const EdgeInsets.all(8.0),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            SizedBox(height: 8),
            Container(
              child: Padding(
                padding: const EdgeInsets.all(8.0),
                child: Icon(icon),
              ),
            ),
            SizedBox(height: 8),
            child,
            Text(title, style: const TextStyle(fontWeight: FontWeight.bold)),
            SizedBox(height: 8),
          ],
        ),
      ),
    );
  }
}
