# Energy Label Extraction Guide for Funda Listings

This guide details the process of extracting energy labels from Funda property listings, using a multi-step approach to ensure reliable extraction.

## Overview

The energy label extraction uses three methods in order of priority:
1. Direct HTML selectors
2. JSON-LD structured data
3. Description text parsing

## 1. Direct HTML Selectors

First, try these CSS selectors in order:

```python
energy_label_selectors = [
    'dt:contains("Energielabel") + dd span::text',  # New format with span
    'dt:contains("Energielabel") + dd div span::text',  # Alternative format
    'dt:contains("Energielabel") + dd::text',  # Old format
    'span[data-test-id="energy-label"]::text',
    'span[class*="energy-label"]::text'
]
```

Process the result:
```python
for selector in energy_label_selectors:
    energy_label = response.css(selector).get()
    if energy_label:
        clean_label = energy_label.strip().upper()
        if re.match(r'^[A-G](\+{1,2})?$', clean_label):
            return clean_label
```

## 2. JSON-LD Data

If HTML selectors fail, try extracting from JSON-LD:

```python
json_ld_scripts = response.css('script[type="application/ld+json"]::text').getall()
for script in json_ld_scripts:
    data = json.loads(script)
    if isinstance(data, dict):
        if 'EnergyData' in str(data) or 'energyLabel' in str(data):
            energy_match = re.search(r'["\']energy(?:Label|Data)["\']\s*:\s*["\']([A-G]\+*)["\']', script, re.IGNORECASE)
            if energy_match:
                return energy_match.group(1).upper()
```

## 3. Description Text

As a last resort, search in the property description:

```python
description = response.css('div.object-description__features li::text, div.object-description-body *::text').getall()
for text in description:
    text = text.strip().lower()
    if 'energielabel' in text or 'energieklasse' in text:
        label_match = re.search(r'energi(?:elabel|eklasse)\s*([a-g](?:\+{1,2})?)', text)
        if label_match:
            return label_match.group(1).upper()
```

## Validation

For all methods, ensure the extracted label:
1. Is a single letter from A to G
2. May have up to two plus signs (e.g., A++, A+)
3. Is converted to uppercase
4. Matches the pattern: `^[A-G](\+{1,2})?$`

## Common Formats

Energy labels appear in these formats:
- Simple: "A", "B", "C", etc.
- With plus: "A+", "A++"
- In text: "Energielabel A", "energieklasse B"

## Error Handling

1. Always validate the extracted label
2. Log failures for debugging
3. Return None or similar if no valid label is found
4. Handle potential JSON parsing errors when working with JSON-LD

## Example Implementation

```python
def extract_energy_label(response):
    # Try HTML selectors
    energy_label_selectors = [
        'dt:contains("Energielabel") + dd span::text',
        'dt:contains("Energielabel") + dd div span::text',
        'dt:contains("Energielabel") + dd::text',
        'span[data-test-id="energy-label"]::text',
        'span[class*="energy-label"]::text'
    ]
    
    for selector in energy_label_selectors:
        if label := response.css(selector).get():
            clean_label = label.strip().upper()
            if re.match(r'^[A-G](\+{1,2})?$', clean_label):
                return clean_label

    # Try JSON-LD
    try:
        for script in response.css('script[type="application/ld+json"]::text').getall():
            data = json.loads(script)
            if isinstance(data, dict):
                if 'EnergyData' in str(data) or 'energyLabel' in str(data):
                    if match := re.search(r'["\']energy(?:Label|Data)["\']\s*:\s*["\']([A-G]\+*)["\']', script, re.IGNORECASE):
                        return match.group(1).upper()
    except (json.JSONDecodeError, AttributeError):
        pass

    # Try description
    description = response.css('div.object-description__features li::text, div.object-description-body *::text').getall()
    for text in description:
        text = text.strip().lower()
        if 'energielabel' in text or 'energieklasse' in text:
            if match := re.search(r'energi(?:elabel|eklasse)\s*([a-g](?:\+{1,2})?)', text):
                return match.group(1).upper()

    return None
```
