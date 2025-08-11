
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/clinical_case_model.dart';
import 'package:ema_educacion_medica_avanzada/core/db/i_local_data.dart';


abstract class IClinicalCaseLocalData implements ILocalData<ClinicalCaseModel>  {}
abstract class IClinicalCaseMessageLocalData implements ILocalData<ChatMessageModel>  {}
