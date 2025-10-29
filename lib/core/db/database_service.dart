import 'package:ema_educacion_medica_avanzada/app/actions/data/local_actions_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/data/local_chat_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/data/local_chat_message_data.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/data/local_clinical_case_data.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:get/get.dart';
import 'package:sqflite/sqflite.dart';


class DatabaseService extends GetxService {
  late final Database db;

  Future<void> _createTables(Database db, int version) async {
    await db.execute(LocalActionsData.sqlInstructionCreateTable());
    await db.execute(LocalChatData.sqlInstructionCreateTable());
    await db.execute(LocalChatMessageData.sqlInstructionCreateTable());
    await db.execute(LocalClinicalCaseData.sqlInstructionCreateTable());
    await db.execute(LocalQuestionsData.sqlInstructionCreateTable());
    await db.execute(LocalQuizzData.sqlInstructionCreateTable());
  }

  Future<void> init() async {
    Logger.log('DatabaseService.init()');

    db = await openDatabase(
      'ema_db_v1_19.db',
      version: 20,
      onCreate: _createTables
    );
  }
}
