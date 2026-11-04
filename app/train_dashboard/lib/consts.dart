import 'dart:ui';

import 'package:flutter/widgets.dart';

class Consts {
static const Color cpBgColor = Color.fromRGBO(27,101,45,1.0);
}


bool isMobile( BuildContext  context) {
 return MediaQuery.of(context).size.width < 600;
}
