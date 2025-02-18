# Making the App City-Agnostic

This document outlines the remaining places in the codebase where Amsterdam-specific code needs to be modified to make the app city-agnostic.

## Frontend Components

### Map Components
1. **PropertyMap.tsx**
   - Hardcoded Amsterdam center coordinates:
   ```typescript
   const AMSTERDAM_CENTER: LatLngTuple = [52.3676, 4.9041];
   ```

2. **PriceHeatmap.tsx**
   - Hardcoded center coordinates:
   ```typescript
   center={[52.3676, 4.9041]}
   ```

## Required Changes

To complete making the app city-agnostic, we need to:

1. **Map Configuration**
   - Make map center and zoom level configurable based on selected city
   - Update map components to use city configuration from the backend

## Implementation Strategy

1. **Phase 1: Frontend Updates**
   - Implement city selection UI
   - Make map components dynamic

2. **Phase 2: Testing**
   - Test the application with different cities
   - Verify all components work correctly with any city 