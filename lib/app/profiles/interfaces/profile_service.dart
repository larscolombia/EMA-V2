import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:image_picker/image_picker.dart';

abstract class ProfileService {
  Future<UserModel> updateProfile(UserModel profile);
  Future<UserModel> updateProfileImage(UserModel profile, XFile imageFile);
  Future<UserModel> fetchDetailedProfile(UserModel profile);
}
