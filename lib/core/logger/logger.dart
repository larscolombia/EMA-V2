// ignore_for_file: avoid_print

class Logger {
  static void log(String message) {
    print(':::::::::::::::::::::::::::::::::::::::::::::::::::::::'); 
    print(':: $message'); 
    print(':::::::::::::::::::::::::::::::::::::::::::::::::::::::'); 
  }

  static void mini(String message) {
    print('::::: $message :::::'); 
  }

  static void error(String message, {String? className, String? methodName, String? meta}) {
    const String red = '\x1B[31m';
    const String reset = '\x1B[0m';
    if (className != null) print('::::: $red $className $reset :::::'); 
    if (methodName != null) print('::::: $red $methodName $reset :::::'); 
    if (meta != null) print('::::: $red $meta $reset :::::'); 
    print('::::: $red $message $reset :::::');  
  }

  static void objectValue(String object, String value) {
    const String blue = '\x1B[34m';
    const String reset = '\x1B[0m';
    // print(':::::::::::::::::::::::::::::::::::::::::::::::::::::::');
    print('OBJECT $blue $object $reset');
    // print('START OF CONTENT');
    log(value);
    // print('END OF CONTENT');
    // print(':::::::::::::::::::::::::::::::::::::::::::::::::::::::');
  }
}