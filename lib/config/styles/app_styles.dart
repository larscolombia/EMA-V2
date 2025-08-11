import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

class AppStyles {
  // Colors
  static const primaryColor = Color.fromRGBO(0, 51, 163, 1);
  static const primary100 = Color.fromRGBO(112, 247, 240, 1);
  static const primary500 = Color.fromRGBO(0, 0, 225, 1);
  static const primary900 = Color.fromRGBO(58, 12, 140, 1);

  static const secondaryColor = Color.fromRGBO(193, 113, 238, 1);
  static const tertiaryColor = Color.fromRGBO(147, 101, 245, 1);

  static const blackColor = Colors.black;
  static const greyColor = Colors.grey;
  static const grey150 = Color.fromRGBO(150, 150, 150, 1);
  static const grey200 = Color.fromRGBO(200, 200, 200, 1);
  static const grey220 = Color.fromRGBO(220, 220, 220, 1);
  static const grey240 = Color.fromRGBO(240, 240, 240, 1);
  static const redColor = Colors.red;
  static const whiteColor = Color.fromRGBO(255, 255, 255, 1); // 245
  static const greenLight = Color.fromRGBO(0, 167, 19, 1);

  // Texts
  static final homeTitle = GoogleFonts.raleway(
    fontWeight: FontWeight.w600,
    fontSize: 30,
    color: primary900,
  );

  static final profileName = GoogleFonts.raleway(
    fontWeight: FontWeight.w300,
    fontSize: 32,
    color: whiteColor,
  );

  static final profileLasName = GoogleFonts.raleway(
    fontWeight: FontWeight.w600,
    fontSize: 32,
    color: whiteColor,
  );

  static final title2 = GoogleFonts.raleway(
    fontWeight: FontWeight.w600,
    fontSize: 24,
    color: primary900,
  );

  static final title2White = GoogleFonts.raleway(
    fontWeight: FontWeight.w600,
    fontSize: 24,
    color: whiteColor,
  );

  static final subtitleBold = GoogleFonts.raleway(
    fontWeight: FontWeight.w600,
    fontSize: 20,
    color: primary900,
  );

  static final subtitle = GoogleFonts.raleway(
    fontWeight: FontWeight.w300,
    fontSize: 20,
    color: primary900,
  );

  static final categoryBtn = GoogleFonts.raleway(
    fontWeight: FontWeight.w600,
    fontSize: 16,
    color: grey150,
  );

  static final menuBtn = GoogleFonts.raleway(
    fontWeight: FontWeight.w600,
    fontSize: 16,
    color: primary900,
  );

  static final breadCrumb = GoogleFonts.raleway(
    fontWeight: FontWeight.w600,
    fontSize: 15,
    color: tertiaryColor,
  );

  static final info1 = GoogleFonts.raleway(
    fontWeight: FontWeight.w600,
    fontSize: 15,
    color: primary900,
  );

  static final info2 = GoogleFonts.raleway(
    fontWeight: FontWeight.w600,
    fontSize: 15,
    color: tertiaryColor,
  );

  static final buttonText = GoogleFonts.raleway(
    fontWeight: FontWeight.w600,
    fontSize: 15,
    color: whiteColor,
  );

  static final chatSubtitle = GoogleFonts.raleway(
    fontWeight: FontWeight.w600,
    fontSize: 24,
    color: whiteColor,
  );

  static final chatMessageAi = GoogleFonts.raleway(
    fontSize: 14,
    color: whiteColor,
  );

  static final chatMessageUser = GoogleFonts.raleway(
    fontSize: 14,
    color: primary900,
  );

  static const inputLabelStyle = TextStyle(
    fontSize: 16.0,
    fontWeight: FontWeight.w600,
    color: Colors.black,
  );

  static final sectionTitle = GoogleFonts.raleway(
    fontWeight: FontWeight.bold,
    fontSize: 18,
    color: primary900,
  );

  static final detailLabel = GoogleFonts.raleway(
    fontWeight: FontWeight.w500,
    fontSize: 14,
    color: grey150,
  );

  static final detailValue = GoogleFonts.raleway(
    fontWeight: FontWeight.w600,
    fontSize: 16,
    color: blackColor,
  );

  static final breadCumb = GoogleFonts.raleway(
    fontWeight: FontWeight.w300,
    fontSize: 12,
    color: blackColor,
  );
}
