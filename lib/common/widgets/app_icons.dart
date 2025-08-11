import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';
import 'package:flutter_svg/svg.dart';

class AppIcons {
  static SvgPicture _baseIcon(String iconName,
      {Color? color, double? height, double? width}) {
    return SvgPicture.asset(
      'assets/icons/svgs/$iconName.svg',
      height: height ?? 28,
      width: width ?? 28,
      colorFilter: ColorFilter.mode(
        color ?? AppStyles.primaryColor,
        BlendMode.srcIn,
      ),
    );
  }

  static SvgPicture arrowLeftSquare(
      {Color? color, double? height, double? width}) {
    return _baseIcon('arrow_left_square',
        color: color, height: height, width: width);
  }

  static SvgPicture arrowRightCircular(
      {Color? color, double? height, double? width}) {
    return _baseIcon('arrow_rigth_circular',
        color: color, height: height, width: width);
  }

  static SvgPicture attachFile({Color? color, double? height, double? width}) {
    return _baseIcon('attach_file', color: color, height: height, width: width);
  }

  static SvgPicture bussinessBag(
      {Color? color, double? height, double? width}) {
    return _baseIcon('bussiness_bag',
        color: color, height: height, width: width);
  }

  static SvgPicture cancel(
      {Color? color, double? height, double? width}) {
    return _baseIcon('cancel',
        color: color, height: height, width: width);
  }

  static SvgPicture chevronRight(
      {Color? color, double? height, double? width}) {
    return _baseIcon('chevron_right',
        color: color, height: height, width: width);
  }

  static SvgPicture clinicalCaseAnalytical(
      {Color? color, double? height, double? width}) {
    return _baseIcon('clinical_case_analytical',
        color: color, height: height, width: width);
  }

  static SvgPicture clinicalCaseInteractive(
      {Color? color, double? height, double? width}) {
    return _baseIcon('clinical_case_interactive',
        color: color, height: height, width: width);
  }

  static SvgPicture closeSquare({Color? color, double? height, double? width}) {
    return _baseIcon('close_square',
        color: color, height: height, width: width);
  }

  static SvgPicture cornerDownLeft(
      {Color? color, double? height, double? width}) {
    return _baseIcon('corner_down_left',
        color: color, height: height, width: width);
  }

  static SvgPicture filePdfSolid(
      {Color? color, double? height, double? width}) {
    return _baseIcon('file_pdf_solid',
        color: color, height: height, width: width);
  }

  static SvgPicture graphicBars({Color? color, double? height, double? width}) {
    return _baseIcon('graphic_bars',
        color: color, height: height, width: width);
  }

  static SvgPicture history({Color? color, double? height, double? width}) {
    return _baseIcon('history', color: color, height: height, width: width);
  }

  static SvgPicture menuSquare({Color? color, double? height, double? width}) {
    return _baseIcon('menu_square', color: color, height: height, width: width);
  }

  static SvgPicture plume({Color? color, double? height, double? width}) {
    return _baseIcon('plume', color: color, height: height, width: width);
  }

  static SvgPicture quizzesEspecial(
      {Color? color, double? height, double? width}) {
    return _baseIcon('quizzes_especial',
        color: color, height: height, width: width);
  }

  static SvgPicture quizzesGeneral(
      {Color? color, double? height, double? width}) {
    return _baseIcon('quizzes_general',
        color: color, height: height, width: width);
  }

  static SvgPicture search({Color? color, double? height, double? width}) {
    return _baseIcon('search', color: color, height: height, width: width);
  }

  static SvgPicture start({Color? color, double? height, double? width}) {
    return _baseIcon('start', color: color, height: height, width: width);
  }

  static SvgPicture startsAi({Color? color, double? height, double? width}) {
    return _baseIcon('stars_ai', color: color, height: height, width: width);
  }

  static SvgPicture userSquare({Color? color, double? height, double? width}) {
    return _baseIcon('user_square', color: color, height: height, width: width);
  }

  static SvgPicture chats({Color? color, double? height, double? width}) {
    return _baseIcon('chats', color: color, height: height, width: width);
  }
}
