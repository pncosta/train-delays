import 'dart:convert';
import 'package:http/http.dart' as http;
import '../models/summary.dart';

class ApiService {

  final String baseUrl;

  const ApiService({this.baseUrl = "http://127.0.0.1:8080"});

  Future<Summary> getSummary() async {
    final uri = Uri.parse('$baseUrl/api/stats/summary');

    final response = await http.get(uri);
    if (response.statusCode >= 300) {
      throw Exception('Server error: ${response.statusCode}');
    }
    final data = json.decode(response.body);
    return Summary.fromJson(data);
  }

  Future<List<Trip>> getWorstTrips() async {
    final uri = Uri.parse('$baseUrl/api/stats/worst');

    final response = await http.get(uri);
    if (response.statusCode >= 300) {
      throw Exception('Server error: ${response.statusCode}');
    }
    final data = json.decode(response.body) as List<dynamic>;
    return data.map((item) => Trip.fromJson(item as Map<String, dynamic>)).toList();
  }

  Future<List<Trip>> getCancellations() async {
    final uri = Uri.parse('$baseUrl/api/stats/cancellations');

    final response = await http.get(uri);
    if (response.statusCode >= 300) {
      throw Exception('Server error: ${response.statusCode}');
    }
    final data = json.decode(response.body) as List<dynamic>;
    return data.map((item) => Trip.fromJson(item as Map<String, dynamic>)).toList();
  }


  Future<List<LeaderboardEntry>> getWorstAverage() async {
    final uri = Uri.parse('$baseUrl/api/stats/worst-average');

    final response = await http.get(uri);
    if (response.statusCode >= 300) {
      throw Exception('Server error: ${response.statusCode}');
    }
    final data = json.decode(response.body) as List<dynamic>;
    return data.map((item) => LeaderboardEntry.fromJson(item as Map<String, dynamic>)).toList();
  }
}
