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
  late Future<List<Trip>> worstFuture;
  late Future<List<LeaderboardEntry>> worstAvgFuture;

  @override
  void initState() {
    super.initState();
    summaryFuture = api.getSummary();
    worstFuture = api.getWorstTrips();
    worstAvgFuture = api.getWorstAverage();
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<Summary>(
      future: summaryFuture,
      builder: (context, summarySnapshot) {
        return FutureBuilder<List<Trip>>(
          future: worstFuture,
          builder: (context, worstSnapshot) {
            return FutureBuilder<List<LeaderboardEntry>>(
              future: worstAvgFuture,
              builder: (context, worstAvgSnapshot) {
                final model = DashboardModel(
                  summary: summarySnapshot,
                  worstDelay: worstSnapshot,
                  worstAvgDelays: worstAvgSnapshot,
                );
                return DashboardContent(model: model);
              },
            );
          },
        );
      },
    );
  }
}

class DashboardModel {
  AsyncSnapshot<Summary> summary;
  AsyncSnapshot<List<Trip>> worstDelay;
  AsyncSnapshot<List<LeaderboardEntry>> worstAvgDelays;

  DashboardModel({
    required this.summary,
    required this.worstDelay,
    required this.worstAvgDelays,
  });
}

class DashboardContent extends StatefulWidget {
  const DashboardContent({
    // required this.summary,
    required this.model,
    super.key,
  });

  // final Summary summary;

  final DashboardModel model;

  @override
  State<DashboardContent> createState() => _DashboardContentState();
}

class _DashboardContentState extends State<DashboardContent> {
  late ServiceTypeStats currentStats;

  @override
  void initState() {
    super.initState();
    _initializeStats();
  }

  void _initializeStats() {
    currentStats =
        widget.model.summary.data?.totalSystem ??
        ServiceTypeStats(serviceType: ServiceType.Total);
  }

  @override
  void didUpdateWidget(DashboardContent oldWidget) {
    super.didUpdateWidget(oldWidget);
    // Rebuild stats when the model changes (i.e., when futures complete)
    if (oldWidget.model != widget.model ||
        oldWidget.model.summary != widget.model.summary) {
      _initializeStats();
    }
  }

  void updateStats(ServiceType serviceType) {
    setState(() {
      currentStats =
          widget.model.summary.data?.breakdown[serviceType] ??
          widget.model.summary.data?.totalSystem ??
          ServiceTypeStats(serviceType: ServiceType.Total);
    });
  }

  @override
  Widget build(BuildContext context) {
    const minSize = 250.0;

    // Check if summary data is still loading
    if (widget.model.summary.connectionState == ConnectionState.waiting) {
      return const Center(child: CircularProgressIndicator());
    }

    if (widget.model.summary.hasError) {
      return Center(
        child: Text('Error loading dashboard: ${widget.model.summary.error}'),
      );
    }

    if (!widget.model.summary.hasData) {
      return const Center(child: Text('No data available'));
    }

    final summary = widget.model.summary.data!;
    final serviceStats = summary.serviceStats;

    return SingleChildScrollView(
      child: Padding(
        padding: const EdgeInsets.all(16.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            // Service Type Filter
            MultipleChoice(onChanged: updateStats),
            // Summary Title
            Text(
              "Estatisticas da última semana",
              style: Theme.of(context).textTheme.displaySmall,
            ),
            const SizedBox(height: 16),
            // Summary cards
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
            const SizedBox(height: 24),
            // Worst trips section
            _buildWorstTripsSection(),
            const SizedBox(height: 24),
            // Worst average section
            _buildWorstAverageSection(),
          ],
        ),
      ),
    );
  }

  Widget _buildWorstTripsSection() {
    final snapshot = widget.model.worstDelay;
    if (snapshot.connectionState == ConnectionState.waiting) {
      return const SizedBox(
        height: 100,
        child: Center(child: CircularProgressIndicator()),
      );
    }

    if (snapshot.hasError) {
      return Center(
        child: Text('Error loading worst trips: ${snapshot.error}'),
      );
    }

    final entries = snapshot.data;
    if (entries == null || entries.isEmpty) {
      return const SizedBox.shrink();
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const Text(
          'Worst Performing Trips',
          style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
        ),
        const SizedBox(height: 12),
        TripsTable(trips: entries),
      ],
    );
  }

  Widget _buildWorstAverageSection() {
    final snapshot = widget.model.worstAvgDelays;
    if (snapshot.connectionState == ConnectionState.waiting) {
      return const SizedBox(
        height: 100,
        child: Center(child: CircularProgressIndicator()),
      );
    }

    if (snapshot.hasError) {
      return Center(
        child: Text('Error loading worst average: ${snapshot.error}'),
      );
    }

    final entries = snapshot.data;
    if (entries == null || entries.isEmpty) {
      return const SizedBox.shrink();
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const Text(
          'Worst Average by Train',
          style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
        ),
        const SizedBox(height: 12),
        LeaderboardEntryTable(entries: entries),
      ],
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
            DataColumn(label: Text('Tipo de Serviço')),
            DataColumn(label: Text('Total')),
            DataColumn(label: Text('Atraso Médio')),
            DataColumn(label: Text('A horas')),
            DataColumn(label: Text('Atrasado')),
            DataColumn(label: Text('Cancelado')),
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

  final ValueChanged<ServiceType>? onChanged;

  @override
  State<MultipleChoice> createState() => _MultipleChoiceState();
}

class _MultipleChoiceState extends State<MultipleChoice> {
  ServiceType selectedType = ServiceType.Total;

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
          children: ServiceType.values.map((ServiceType serviceType) {
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
            const SizedBox(height: 8),
            Padding(padding: const EdgeInsets.all(8.0), child: Icon(icon)),
            const SizedBox(height: 8),
            child,
            Text(title, style: const TextStyle(fontWeight: FontWeight.bold)),
            const SizedBox(height: 8),
          ],
        ),
      ),
    );
  }
}

class TripsTable extends StatelessWidget {
  final List<Trip> trips;

  const TripsTable({required this.trips, super.key});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: DataTable(
          columns: const [
            DataColumn(label: Text('No. Comboio')),
            DataColumn(label: Text('Tipo de Serviço')),
            DataColumn(label: Text('Percurso')),
            DataColumn(label: Text('Atraso')),
            DataColumn(label: Text('Estado')),
          ],
          rows: trips.map((trip) {
            return DataRow(
              cells: [
                DataCell(Text(trip.trainNumber)),
                DataCell(Text(trip.serviceType.toLocalizedString(context))),
                DataCell(
                  Text('${trip.originStation} → ${trip.destinationStation}'),
                ),
                DataCell(Text('${trip.delayMinutes ?? 0} min')),
                DataCell(
                  Text(
                    trip.isCancelled == true
                        ? 'Cancelado'
                        : trip.isDelayed
                        ? 'Atrasado'
                        : 'A horas',
                    style: TextStyle(
                      color: trip.isCancelled == true
                          ? Colors.redAccent
                          : trip.isDelayed ? Colors.orangeAccent : Colors.greenAccent,
                    ),
                  ),
                ),
              ],
            );
          }).toList(),
        ),
      ),
    );
  }
}

class LeaderboardEntryTable extends StatelessWidget {
  final List<LeaderboardEntry> entries;
  final String unitLabel;

  const LeaderboardEntryTable({
    required this.entries,
    this.unitLabel = "Average Value",
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: DataTable(
          columns: [
            DataColumn(label: Text('No.Comboio')),
            DataColumn(label: Text('Tipo de Serviço')),
            DataColumn(label: Text('Percurso')),
            DataColumn(label: Text(unitLabel)),
            DataColumn(label: Text('No. Viagens')),
          ],
          rows: entries.map((entry) {
            return DataRow(
              cells: [
                DataCell(Text(entry.trainNumber)),
                DataCell(Text(entry.serviceType.toLocalizedString(context))),
                DataCell(
                  Text('${entry.originStation} → ${entry.destinationStation}'),
                ),
                DataCell(Text(entry.value.toStringAsFixed(2))),
                DataCell(Text(entry.count.toString())),
              ],
            );
          }).toList(),
        ),
      ),
    );
  }
}
