# Metropolitan Areas Implementation

## Overview
This document outlines the implementation of metropolitan areas in the property tracking system. The system allows users to define and manage metropolitan areas (e.g., Amsterdam metropolitan area including Amstelveen and Diemen) through a user-friendly frontend interface.

## Implementation Status

### Completed
1. ✅ Backend database schema and migrations
2. ✅ REST API endpoints for CRUD operations
3. ✅ Data model and type definitions
4. ✅ Basic metropolitan area management

### Next Phase: Frontend Implementation
The frontend implementation will consist of the following components:

1. **Metropolitan Area Management Page**
   ```typescript
   interface MetropolitanArea {
       id: number;
       name: string;
       cities: string[];
   }
   ```

   Components needed:
   - MetropolitanAreaList: Display all metropolitan areas
   - MetropolitanAreaForm: Create/edit metropolitan areas
   - CitySelector: Multi-select component for choosing cities
   - ConfirmationDialog: For delete operations

2. **Integration with Property Analysis**
   - Update property filters to include metropolitan area selection
   - Extend statistics calculations to metropolitan areas
   - Add metropolitan area overlay to the map visualization

## Frontend Components Design

### MetropolitanAreaList
```typescript
const MetropolitanAreaList: React.FC = () => {
    const [areas, setAreas] = useState<MetropolitanArea[]>([]);
    // Fetch and display metropolitan areas
    // Include edit/delete actions
};
```

### MetropolitanAreaForm
```typescript
interface MetropolitanAreaFormProps {
    area?: MetropolitanArea;
    onSubmit: (area: MetropolitanArea) => void;
}

const MetropolitanAreaForm: React.FC<MetropolitanAreaFormProps> = () => {
    // Form for creating/editing metropolitan areas
    // Include city selection
};
```

### CitySelector
```typescript
interface CitySelectorProps {
    selectedCities: string[];
    onChange: (cities: string[]) => void;
}

const CitySelector: React.FC<CitySelectorProps> = () => {
    // Multi-select component for cities
    // Autocomplete support
};
```

## API Integration

### Metropolitan Area Service
```typescript
// services/metropolitan.ts
export const metropolitanApi = {
    getAll: () => axios.get<MetropolitanArea[]>('/api/metropolitan'),
    getOne: (name: string) => axios.get<MetropolitanArea>(`/api/metropolitan/${name}`),
    create: (area: MetropolitanArea) => axios.post('/api/metropolitan', area),
    update: (name: string, area: MetropolitanArea) => 
        axios.put(`/api/metropolitan/${name}`, area),
    delete: (name: string) => axios.delete(`/api/metropolitan/${name}`)
};
```

## User Interface Design

1. **List View**
   - Display metropolitan areas in a Material-UI table
   - Show name and number of cities
   - Actions: Edit, Delete
   - "Create New" button

2. **Form View**
   - Name field (required)
   - City selector (multi-select with autocomplete)
   - Save/Cancel buttons
   - Validation feedback

3. **Integration Points**
   - Add metropolitan area filter to property search
   - Include metropolitan areas in statistics views
   - Show metropolitan boundaries on map

## Implementation Steps

1. Create new route and page component
2. Implement metropolitan area service
3. Build form components
4. Add list view with CRUD operations
5. Integrate with existing property filters
6. Add metropolitan area statistics
7. Update map visualization

## Testing Strategy

1. **Unit Tests**
   - Form validation
   - Component rendering
   - Service functions

2. **Integration Tests**
   - CRUD operations
   - API integration
   - Filter functionality

3. **End-to-End Tests**
   - Complete metropolitan area management workflow
   - Integration with property search and statistics

## Next Steps

1. Create new frontend route `/metropolitan-areas`
2. Implement basic CRUD components
3. Add metropolitan area selection to filters
4. Update property statistics to support metropolitan areas
5. Add metropolitan area boundaries to map visualization

## Notes

- Use Material-UI components for consistency
- Implement proper error handling and loading states
- Add confirmation dialogs for destructive actions
- Include proper form validation
- Ensure responsive design for all components

## Core Concept
Instead of using database schemas or complex data structures, we'll use a simple JSON configuration file where:
- Metropolitan areas are defined in a configuration file
- Each metropolitan area is a simple list of constituent cities
- No database changes or schemas are required
- Configuration can be updated through the frontend

## Configuration Structure

### Simple Metropolitan Area Configuration
```json
{
    "metropolitan_areas": [
        {
            "name": "Amsterdam Metro",
            "cities": ["Amsterdam", "Amstelveen", "Diemen"]
        },
        {
            "name": "Rotterdam Metro",
            "cities": ["Rotterdam", "Schiedam", "Vlaardingen"]
        }
    ]
}
```

## Implementation Strategy

1. **Configuration Management**
   - Store metropolitan configurations in a JSON file
   - Load configurations at runtime
   - Save updates through file operations
   - No database schema changes needed

2. **Query Handling**
   - When querying for a metropolitan area:
     - Read the configuration file
     - Include all properties from constituent cities
     - Maintain original city names in the database
     - Aggregate statistics across all included cities

3. **Spider Integration**
   - When scraping a metropolitan area:
     - Run spider for each constituent city
     - Maintain original city names in the data
     - No need for postal code management

## Benefits
1. Simple to implement and maintain
2. Flexible - cities can be added or removed easily
3. No database schema changes required
4. Clear and understandable for users
5. No need for complex postal code management
6. Easy to extend to other metropolitan areas

## Example Usage

### Frontend Interface
```typescript
interface MetropolitanAreaConfig {
    name: string;
    cities: string[];
}

// Example configuration
const amsterdamMetro: MetropolitanAreaConfig = {
    name: "Amsterdam Metropolitan Area",
    cities: ["Amsterdam", "Amstelveen", "Diemen"]
};
```

### Query Handling
- When user selects "Amsterdam Metropolitan Area":
  - Backend automatically includes properties from all constituent cities
  - Statistics are aggregated across all included cities
  - Original city names are preserved in the data

## Integration Points

### API Changes
- New endpoints for:
  - Managing metropolitan area definitions
  - Querying metropolitan area data
  - Getting aggregated statistics

### Spider Integration
- Spider manager handles multiple cities sequentially
- No changes needed to spider logic
- Cities maintain their original identities

## Limitations
1. Relies on user configuration
2. Manual updates required when metropolitan area composition changes
3. No automatic geographic validation

## Next Steps
1. Design metropolitan area management UI
2. Implement metropolitan area configuration endpoints
3. Update query handling to support metropolitan areas
4. Add metropolitan area support to statistics calculations 