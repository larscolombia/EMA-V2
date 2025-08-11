import 'package:flutter/material.dart';

class SlideInLeft extends StatefulWidget {
  final Widget child;
  final Duration duration;

  const SlideInLeft({
    super.key,
    required this.child,
    this.duration = const Duration(milliseconds: 200),
  });

  @override
  State<SlideInLeft> createState() => _SlideInLeftState();
}

class _SlideInLeftState extends State<SlideInLeft>
    with SingleTickerProviderStateMixin {
  late AnimationController _controller;
  late Animation<Offset> _offsetAnimation;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      duration: widget.duration,
      vsync: this,
    );

    _offsetAnimation = Tween<Offset>(
      begin: const Offset(-0.5, 0.0),
      end: Offset.zero,
    ).animate(CurvedAnimation(
      parent: _controller,
      curve: Curves.easeOut,
    ));

    _controller.forward();
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return SlideTransition(
      position: _offsetAnimation,
      child: FadeTransition(
        opacity: _controller,
        child: widget.child,
      ),
    );
  }
}
