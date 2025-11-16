# QuantaraX Flutter Architecture

This Flutter project follows **Clean Architecture** principles with a modular, scalable structure suitable for industrial-level development.

## 📁 Project Structure

```
lib/
├── main.dart                          # App entry point
├── quantarax.dart                     # Main barrel export file
├── config/                            # App-wide configuration
│   └── app_theme.dart                # Theme and styling configuration
├── core/                              # Core business logic (framework independent)
│   ├── constants/                     # App-wide constants
│   │   ├── app_constants.dart        # General app constants
│   │   └── route_constants.dart      # Navigation route constants
│   ├── errors/                        # Error handling
│   │   └── exceptions.dart           # Custom exception classes
│   ├── services/                      # Core business services
│   │   └── file_service.dart         # File handling service
│   ├── utils/                         # Utility classes
│   │   └── responsive_utils.dart     # Responsive design utilities
│   └── network/                       # Network layer (future)
├── data/                              # Data layer
│   ├── models/                        # Data models
│   │   ├── chat_model.dart           # Chat data model
│   │   └── transfer_model.dart       # Transfer data model
│   ├── repositories/                  # Repository implementations
│   └── datasources/                   # Data sources (API, local storage)
├── domain/                            # Domain layer (business rules)
│   ├── entities/                      # Business entities
│   ├── repositories/                  # Repository contracts
│   └── usecases/                      # Business use cases
└── presentation/                      # Presentation layer (UI)
    ├── pages/                         # Screen/page widgets
    │   ├── desktop_page.dart         # Desktop responsive layout
    │   └── mobile_pages.dart         # Mobile screen layouts
    ├── widgets/                       # Reusable UI components
    │   ├── widgets.dart              # Barrel export for widgets
    │   ├── common/                    # Common/shared widgets
    │   │   └── custom_widgets.dart   # Custom reusable widgets
    │   ├── chat/                      # Chat-specific widgets
    │   │   └── chat_panel.dart       # Chat interface widget
    │   ├── sidebar/                   # Sidebar-specific widgets
    │   │   └── sidebar_widget.dart   # Navigation sidebar
    │   └── monitoring/                # Monitoring-specific widgets
    │       └── monitoring_panel.dart # Transfer monitoring widget
    ├── providers/                     # State management (future)
    └── controllers/                   # Business logic controllers (future)
```

## 🏗️ Architecture Principles

### **1. Clean Architecture Layers**
- **Presentation Layer**: UI components, state management, user interactions
- **Domain Layer**: Business logic, entities, use cases (framework independent)
- **Data Layer**: Models, repositories, data sources, external API calls

### **2. Separation of Concerns**
- Each layer has a single responsibility
- Dependencies point inward (Dependency Inversion Principle)
- Business logic is isolated from UI and external dependencies

### **3. Modular Structure**
- Feature-based organization within each layer
- Clear boundaries between different functional areas
- Easy to maintain, test, and scale

## 📱 Responsive Design

### **Breakpoints**
- **Mobile**: < 768px - Stack navigation with individual screens
- **Tablet**: 768px - 1200px - Two-panel layout with toggleable monitoring
- **Desktop**: ≥ 1200px - Three-panel layout with all components visible

### **Layout Strategy**
- **Mobile**: `Navigator` with stack-based screen routing
- **Tablet**: `Row` with conditional panel visibility
- **Desktop**: `Row` with all panels and toggle functionality

## 🧩 Key Components

### **Core Services**
- **FileService**: Handle file picking, validation, and formatting
- **ResponsiveUtils**: Responsive design utilities and breakpoint management

### **Data Models**
- **ChatModel**: Chat/conversation data structure
- **TransferModel**: File transfer data structure with progress tracking

### **UI Widgets**
- **Responsive Widgets**: Adapt behavior based on screen size
- **Reusable Components**: Consistent design system implementation
- **Feature-Specific**: Organized by functional area

## 🚀 Getting Started

### **Import Structure**
```dart
// Main library export (recommended)
import 'package:quantarax/quantarax.dart';

// Specific module imports
import 'package:quantarax/presentation/widgets/widgets.dart';
import 'package:quantarax/core/constants/app_constants.dart';
```

### **Adding New Features**
1. **Create Models** in `data/models/`
2. **Define Repository Contracts** in `domain/repositories/`
3. **Implement Use Cases** in `domain/usecases/`
4. **Create UI Widgets** in appropriate `presentation/widgets/` subfolder
5. **Add Pages** in `presentation/pages/`
6. **Export** in relevant barrel files

### **State Management** (Future Implementation)
- **Provider**: For simple state management
- **Bloc/Cubit**: For complex state management
- **Riverpod**: Alternative state management solution

## 🔧 Development Guidelines

### **Code Organization**
- Use barrel exports for clean imports
- Follow consistent naming conventions
- Group related functionality together
- Keep widgets focused and small

### **Responsive Development**
- Use `ResponsiveUtils` for breakpoint logic
- Test on multiple screen sizes
- Implement adaptive layouts, not just responsive

### **Error Handling**
- Use custom exceptions from `core/errors/`
- Implement proper error boundaries
- Provide meaningful error messages

## 📦 Dependencies

### **Core Dependencies**
- `flutter`: Flutter framework
- `flutter_svg`: SVG support
- `file_picker`: File selection
- `equatable`: Value equality

### **UI Dependencies**
- `gradient_borders`: Gradient borders
- `simple_gradient_text`: Gradient text
- `shimmer`: Loading animations
- `fl_chart`: Charts and graphs

This architecture provides a solid foundation for building scalable, maintainable Flutter applications that can grow with your project's needs.