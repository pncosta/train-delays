import 'package:flutter/material.dart';

class InfoDialog extends StatelessWidget {
  const InfoDialog({super.key});

  @override
  Widget build(BuildContext context) {
    return SimpleDialog(
      title: Text('Sobre'),
      contentPadding: EdgeInsets.all(16.0),
      children: [
        Text('Informação proveniente da CP - Comboios de Portugal E.P.E.'),
        SizedBox(height: 4),
        Text(
          'Este website não tem qualquer ligação à CP E.P.E e foi desenolvido apenas por motivos lúdicos,não garantindo que a informação apresentada seja correta.',
        ),
        SizedBox(height: 4),
        Text('Só as autoridades competentes poderão fornecer dados oficiais'),
        SizedBox(height: 8),
        SimpleDialogOption(
          onPressed: () {
            Navigator.pop(context);
          },
          child: const Text('FECHAR'),
        ),
      ],
    );
  }
}
