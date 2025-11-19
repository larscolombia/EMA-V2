import 'package:ema_educacion_medica_avanzada/admin/shared/constants/admin_colors.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/layout/admin_layout.dart';
import 'package:ema_educacion_medica_avanzada/admin/shared/widgets/cards/metric_card.dart';
import 'package:ema_educacion_medica_avanzada/admin/features/plans/controllers/plans_controller.dart';
import 'package:ema_educacion_medica_avanzada/admin/features/plans/models/plan.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:get/get.dart';

class PlansPage extends StatelessWidget {
  const PlansPage({super.key});

  @override
  Widget build(BuildContext context) {
    final controller = Get.put(PlansController());

    return AdminLayout(
      title: 'Gestión de Planes',
      child: Obx(() {
        if (controller.isLoading.value && controller.plans.isEmpty) {
          return const Center(child: CircularProgressIndicator());
        }

        final filteredPlans = controller.filteredPlans;

        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Métricas
            Padding(
              padding: const EdgeInsets.all(24),
              child: Row(
                children: [
                  Expanded(
                    child: MetricCard(
                      title: 'Total Planes',
                      value: controller.plans.length.toString(),
                      icon: Icons.credit_card,
                      color: AdminColors.primary,
                      subtitle: 'Planes disponibles',
                    ),
                  ),
                  const SizedBox(width: 16),
                  Expanded(
                    child: MetricCard(
                      title: 'Planes de Pago',
                      value:
                          controller.plans
                              .where((p) => p.price > 0)
                              .length
                              .toString(),
                      icon: Icons.attach_money,
                      color: Colors.green,
                      subtitle: 'Con precio',
                    ),
                  ),
                  const SizedBox(width: 16),
                  Expanded(
                    child: MetricCard(
                      title: 'Planes Gratis',
                      value:
                          controller.plans
                              .where((p) => p.price == 0)
                              .length
                              .toString(),
                      icon: Icons.card_giftcard,
                      color: Colors.orange,
                      subtitle: 'Sin costo',
                    ),
                  ),
                ],
              ),
            ),

            // Header con búsqueda y botón crear
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 24),
              child: Row(
                children: [
                  Expanded(
                    flex: 2,
                    child: TextField(
                      decoration: InputDecoration(
                        hintText: 'Buscar planes...',
                        prefixIcon: const Icon(Icons.search),
                        border: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(12),
                        ),
                        filled: true,
                        fillColor: Colors.grey[50],
                      ),
                      onChanged: controller.updateSearchQuery,
                    ),
                  ),
                  const SizedBox(width: 16),
                  ElevatedButton.icon(
                    onPressed: () => _showPlanDialog(context, controller),
                    icon: const Icon(Icons.add),
                    label: const Text('Nuevo Plan'),
                    style: ElevatedButton.styleFrom(
                      backgroundColor: AdminColors.primary,
                      foregroundColor: Colors.white,
                      padding: const EdgeInsets.symmetric(
                        horizontal: 24,
                        vertical: 16,
                      ),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(12),
                      ),
                    ),
                  ),
                ],
              ),
            ),

            const SizedBox(height: 24),

            // Tabla de planes
            Expanded(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Container(
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(16),
                    boxShadow: [
                      BoxShadow(
                        color: Colors.black.withValues(alpha: 0.05),
                        blurRadius: 10,
                        offset: const Offset(0, 2),
                      ),
                    ],
                  ),
                  child: Column(
                    children: [
                      if (filteredPlans.isEmpty)
                        const Padding(
                          padding: EdgeInsets.all(48),
                          child: Column(
                            children: [
                              Icon(
                                Icons.credit_card_off,
                                size: 64,
                                color: Colors.grey,
                              ),
                              SizedBox(height: 16),
                              Text(
                                'No se encontraron planes',
                                style: TextStyle(
                                  fontSize: 16,
                                  color: Colors.grey,
                                ),
                              ),
                            ],
                          ),
                        )
                      else
                        Expanded(
                          child: SingleChildScrollView(
                            child: SizedBox(
                              width: double.infinity,
                              child: DataTable(
                                headingRowColor: WidgetStateProperty.all(
                                  AdminColors.primary.withValues(alpha: 0.1),
                                ),
                                columnSpacing: 24,
                                horizontalMargin: 24,
                                dataRowMinHeight: 60,
                                dataRowMaxHeight: 70,
                                columns: const [
                                  DataColumn(
                                    label: Text(
                                      'ID',
                                      style: TextStyle(
                                        fontWeight: FontWeight.bold,
                                      ),
                                    ),
                                  ),
                                  DataColumn(
                                    label: Text(
                                      'Plan',
                                      style: TextStyle(
                                        fontWeight: FontWeight.bold,
                                      ),
                                    ),
                                  ),
                                  DataColumn(
                                    label: Text(
                                      'Precio',
                                      style: TextStyle(
                                        fontWeight: FontWeight.bold,
                                      ),
                                    ),
                                  ),
                                  DataColumn(
                                    label: Text(
                                      'Frecuencia',
                                      style: TextStyle(
                                        fontWeight: FontWeight.bold,
                                      ),
                                    ),
                                  ),
                                  DataColumn(
                                    label: Text(
                                      'Consultas',
                                      style: TextStyle(
                                        fontWeight: FontWeight.bold,
                                      ),
                                    ),
                                  ),
                                  DataColumn(
                                    label: Text(
                                      'Casos',
                                      style: TextStyle(
                                        fontWeight: FontWeight.bold,
                                      ),
                                    ),
                                  ),
                                  DataColumn(
                                    label: Text(
                                      'Cuestionarios',
                                      style: TextStyle(
                                        fontWeight: FontWeight.bold,
                                      ),
                                    ),
                                  ),
                                  DataColumn(
                                    label: Text(
                                      'Acciones',
                                      style: TextStyle(
                                        fontWeight: FontWeight.bold,
                                      ),
                                    ),
                                  ),
                                ],
                                rows:
                                    filteredPlans.map((plan) {
                                      return DataRow(
                                        cells: [
                                          DataCell(Text('#${plan.id}')),
                                          DataCell(
                                            Row(
                                              children: [
                                                Icon(
                                                  plan.price > 0
                                                      ? Icons.star
                                                      : Icons.card_giftcard,
                                                  size: 16,
                                                  color:
                                                      plan.price > 0
                                                          ? Colors.amber
                                                          : Colors.grey,
                                                ),
                                                const SizedBox(width: 8),
                                                Text(
                                                  plan.name,
                                                  style: const TextStyle(
                                                    fontWeight: FontWeight.w600,
                                                  ),
                                                ),
                                              ],
                                            ),
                                          ),
                                          DataCell(
                                            Text(
                                              plan.formattedPrice,
                                              style: TextStyle(
                                                fontWeight: FontWeight.w600,
                                                color:
                                                    plan.price > 0
                                                        ? Colors.green[700]
                                                        : Colors.grey,
                                              ),
                                            ),
                                          ),
                                          DataCell(
                                            Container(
                                              padding:
                                                  const EdgeInsets.symmetric(
                                                    horizontal: 12,
                                                    vertical: 6,
                                                  ),
                                              decoration: BoxDecoration(
                                                color:
                                                    plan.billing == 'Mensual'
                                                        ? Colors.blue
                                                            .withValues(
                                                              alpha: 0.1,
                                                            )
                                                        : Colors.purple
                                                            .withValues(
                                                              alpha: 0.1,
                                                            ),
                                                borderRadius:
                                                    BorderRadius.circular(8),
                                              ),
                                              child: Text(
                                                plan.billing,
                                                style: TextStyle(
                                                  color:
                                                      plan.billing == 'Mensual'
                                                          ? Colors.blue[700]
                                                          : Colors.purple[700],
                                                  fontSize: 12,
                                                  fontWeight: FontWeight.w600,
                                                ),
                                              ),
                                            ),
                                          ),
                                          DataCell(
                                            Text(
                                              plan.consultations.toString(),
                                              style: const TextStyle(
                                                fontWeight: FontWeight.w500,
                                              ),
                                            ),
                                          ),
                                          DataCell(
                                            Text(
                                              plan.clinicalCases.toString(),
                                              style: const TextStyle(
                                                fontWeight: FontWeight.w500,
                                              ),
                                            ),
                                          ),
                                          DataCell(
                                            Text(
                                              plan.questionnaires.toString(),
                                              style: const TextStyle(
                                                fontWeight: FontWeight.w500,
                                              ),
                                            ),
                                          ),
                                          DataCell(
                                            Row(
                                              mainAxisSize: MainAxisSize.min,
                                              children: [
                                                IconButton(
                                                  icon: const Icon(
                                                    Icons.edit,
                                                    size: 20,
                                                  ),
                                                  onPressed:
                                                      () => _showPlanDialog(
                                                        context,
                                                        controller,
                                                        plan: plan,
                                                      ),
                                                  tooltip: 'Editar',
                                                  color: AdminColors.primary,
                                                ),
                                                IconButton(
                                                  icon: const Icon(
                                                    Icons.delete,
                                                    size: 20,
                                                  ),
                                                  onPressed:
                                                      () => _confirmDelete(
                                                        controller,
                                                        plan,
                                                      ),
                                                  tooltip: 'Eliminar',
                                                  color: Colors.red[400],
                                                ),
                                              ],
                                            ),
                                          ),
                                        ],
                                      );
                                    }).toList(),
                              ),
                            ),
                          ),
                        ),
                    ],
                  ),
                ),
              ),
            ),
          ],
        );
      }),
    );
  }

  void _showPlanDialog(
    BuildContext context,
    PlansController controller, {
    Plan? plan,
  }) {
    final isEditing = plan != null;
    final nameController = TextEditingController(text: plan?.name ?? '');
    final priceController = TextEditingController(
      text: plan?.price.toString() ?? '0',
    );
    final consultationsController = TextEditingController(
      text: plan?.consultations.toString() ?? '0',
    );
    final questionnairesController = TextEditingController(
      text: plan?.questionnaires.toString() ?? '0',
    );
    final casesController = TextEditingController(
      text: plan?.clinicalCases.toString() ?? '0',
    );
    final filesController = TextEditingController(
      text: plan?.files.toString() ?? '0',
    );
    final billingRx = Rx<String>(plan?.billing ?? 'Mensual');
    final currencyRx = Rx<String>(plan?.currency ?? 'USD');

    Get.dialog(
      Dialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
        child: Container(
          width: 600,
          padding: const EdgeInsets.all(24),
          child: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  isEditing ? 'Editar Plan' : 'Nuevo Plan',
                  style: const TextStyle(
                    fontSize: 24,
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const SizedBox(height: 24),
                TextField(
                  controller: nameController,
                  decoration: InputDecoration(
                    labelText: 'Nombre del Plan',
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(12),
                    ),
                  ),
                ),
                const SizedBox(height: 16),
                Row(
                  children: [
                    Expanded(
                      child: TextField(
                        controller: priceController,
                        decoration: InputDecoration(
                          labelText: 'Precio',
                          border: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(12),
                          ),
                        ),
                        keyboardType: const TextInputType.numberWithOptions(
                          decimal: true,
                        ),
                        inputFormatters: [
                          FilteringTextInputFormatter.allow(
                            RegExp(r'^\d+\.?\d{0,2}'),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(width: 16),
                    Expanded(
                      child: Obx(
                        () => DropdownButtonFormField<String>(
                          value: currencyRx.value,
                          decoration: InputDecoration(
                            labelText: 'Moneda',
                            border: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(12),
                            ),
                          ),
                          items:
                              ['USD', 'EUR', 'MXN'].map((currency) {
                                return DropdownMenuItem(
                                  value: currency,
                                  child: Text(currency),
                                );
                              }).toList(),
                          onChanged: (value) => currencyRx.value = value!,
                        ),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 16),
                Obx(
                  () => DropdownButtonFormField<String>(
                    value: billingRx.value,
                    decoration: InputDecoration(
                      labelText: 'Frecuencia de Pago',
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(12),
                      ),
                    ),
                    items:
                        ['Mensual', 'Anual'].map((billing) {
                          return DropdownMenuItem(
                            value: billing,
                            child: Text(billing),
                          );
                        }).toList(),
                    onChanged: (value) => billingRx.value = value!,
                  ),
                ),
                const SizedBox(height: 16),
                Row(
                  children: [
                    Expanded(
                      child: TextField(
                        controller: consultationsController,
                        decoration: InputDecoration(
                          labelText: 'Consultas',
                          border: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(12),
                          ),
                        ),
                        keyboardType: TextInputType.number,
                        inputFormatters: [
                          FilteringTextInputFormatter.digitsOnly,
                        ],
                      ),
                    ),
                    const SizedBox(width: 16),
                    Expanded(
                      child: TextField(
                        controller: casesController,
                        decoration: InputDecoration(
                          labelText: 'Casos Clínicos',
                          border: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(12),
                          ),
                        ),
                        keyboardType: TextInputType.number,
                        inputFormatters: [
                          FilteringTextInputFormatter.digitsOnly,
                        ],
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 16),
                Row(
                  children: [
                    Expanded(
                      child: TextField(
                        controller: questionnairesController,
                        decoration: InputDecoration(
                          labelText: 'Cuestionarios',
                          border: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(12),
                          ),
                        ),
                        keyboardType: TextInputType.number,
                        inputFormatters: [
                          FilteringTextInputFormatter.digitsOnly,
                        ],
                      ),
                    ),
                    const SizedBox(width: 16),
                    Expanded(
                      child: TextField(
                        controller: filesController,
                        decoration: InputDecoration(
                          labelText: 'Archivos',
                          border: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(12),
                          ),
                        ),
                        keyboardType: TextInputType.number,
                        inputFormatters: [
                          FilteringTextInputFormatter.digitsOnly,
                        ],
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 24),
                Row(
                  mainAxisAlignment: MainAxisAlignment.end,
                  children: [
                    TextButton(
                      onPressed: () => Get.back(),
                      child: const Text('Cancelar'),
                    ),
                    const SizedBox(width: 12),
                    ElevatedButton(
                      onPressed: () {
                        final newPlan = Plan(
                          id: plan?.id,
                          name: nameController.text,
                          price: double.tryParse(priceController.text) ?? 0,
                          currency: currencyRx.value,
                          billing: billingRx.value,
                          consultations:
                              int.tryParse(consultationsController.text) ?? 0,
                          questionnaires:
                              int.tryParse(questionnairesController.text) ?? 0,
                          clinicalCases:
                              int.tryParse(casesController.text) ?? 0,
                          files: int.tryParse(filesController.text) ?? 0,
                        );

                        if (isEditing) {
                          controller.updatePlan(newPlan);
                        } else {
                          controller.createPlan(newPlan);
                        }
                      },
                      style: ElevatedButton.styleFrom(
                        backgroundColor: AdminColors.primary,
                        foregroundColor: Colors.white,
                      ),
                      child: Text(isEditing ? 'Actualizar' : 'Crear'),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  void _confirmDelete(PlansController controller, Plan plan) {
    Get.dialog(
      AlertDialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
        title: Row(
          children: [
            Icon(
              Icons.warning_amber_rounded,
              color: Colors.orange[700],
              size: 28,
            ),
            const SizedBox(width: 12),
            const Text('Confirmar eliminación'),
          ],
        ),
        content: Text(
          '¿Estás seguro de que deseas eliminar el plan "${plan.name}"? Esta acción no se puede deshacer.',
        ),
        actions: [
          TextButton(
            onPressed: () => Get.back(),
            child: const Text('Cancelar'),
          ),
          ElevatedButton(
            onPressed: () {
              Get.back();
              controller.deletePlan(plan.id!);
            },
            style: ElevatedButton.styleFrom(
              backgroundColor: Colors.red,
              foregroundColor: Colors.white,
            ),
            child: const Text('Eliminar'),
          ),
        ],
      ),
    );
  }
}
