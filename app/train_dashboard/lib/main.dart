import 'package:flutter/material.dart';
import 'package:train_dashboard/pages/dashboard.dart';
import 'package:train_dashboard/pages/info_dialog.dart';
import 'package:url_launcher/url_launcher.dart';

import 'consts.dart';

void main() {
  runApp(MaterialApp(home: const MainApp()));
}

class MainApp extends StatelessWidget {
  const MainApp({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        backgroundColor: Consts.cpBgColor,
        title: const Text(
          'Estatisticas dos Comboios de Portugal',
          style: TextStyle(color: Colors.white, fontSize: 20.0),
        ),
        actions: <Widget>[
          IconButton(
            icon: const Icon(Icons.info),
            color: Colors.white,
            tooltip: 'Information',
            onPressed: () {
              showDialog(context: context, builder: (context) => InfoDialog());
            },
          ),

          IconButton(
            icon: const Icon(Icons.code),
            tooltip: 'github',
            color: Colors.white,
            onPressed: () {
              launchUrl(Uri.parse('https://github.com/pncosta'));
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(content: Text('This is a snackbar')),
              );
            },
          ),
        ],
      ),
      body: Center(child: Dashboard()),
    );
  }
}
