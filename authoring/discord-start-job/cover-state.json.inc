    "desiredCoverState": {
      "type": "object",
      "additionalProperties": false,
      "required": ["active", "revision", "source"],
      "properties": {
        "active": {
          "type": "boolean"
        },
        "revision": {
          "type": "integer",
          "minimum": 1
        },
        "source": { "enum": ["none", "preset", "upload"] },
        "variant_id": {
          "type": "string",
          "minLength": 1,
          "maxLength": 128,
          "pattern": "^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$"
        }
      },
      "allOf": [
        {
          "if": {
            "required": ["active"],
            "properties": { "active": { "const": true } }
          },
          "then": {
            "required": ["variant_id"],
            "properties": { "source": { "enum": ["preset", "upload"] } }
          },
          "else": {
            "properties": { "source": { "const": "none" } },
            "not": { "required": ["variant_id"] }
          }
        }
      ]
    },
    "appliedCoverState": {
      "type": "object",
      "additionalProperties": false,
      "required": ["state"],
      "properties": {
        "state": {
          "enum": ["known", "unknown"]
        },
        "active": {
          "type": "boolean"
        },
        "revision": {
          "type": "integer",
          "minimum": 1
        },
        "variant_id": {
          "type": "string",
          "minLength": 1,
          "maxLength": 128,
          "pattern": "^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$"
        }
      },
      "allOf": [
        {
          "if": {
            "required": ["state"],
            "properties": {
              "state": {
                "const": "known"
              }
            }
          },
          "then": {
            "required": ["active", "revision"]
          },
          "else": {
            "not": {
              "anyOf": [
                {
                  "required": ["active"]
                },
                {
                  "required": ["revision"]
                }
              ]
            }
          }
        },
        {
          "if": {
            "required": ["state", "active"],
            "properties": {
              "state": { "const": "known" },
              "active": { "const": true }
            }
          },
          "then": { "required": ["variant_id"] },
          "else": { "not": { "required": ["variant_id"] } }
        }
      ]
    },
    "knownAppliedCoverState": {
      "type": "object",
      "additionalProperties": false,
      "required": ["state", "active", "revision"],
      "properties": {
        "state": { "const": "known" },
        "active": { "type": "boolean" },
        "revision": { "type": "integer", "minimum": 1 },
        "variant_id": {
          "type": "string",
          "minLength": 1,
          "maxLength": 128,
          "pattern": "^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$"
        }
      },
      "allOf": [
        {
          "if": {
            "required": ["active"],
            "properties": { "active": { "const": true } }
          },
          "then": { "required": ["variant_id"] },
          "else": { "not": { "required": ["variant_id"] } }
        }
      ]
    },
    "layerState": {
      "type": "object",
      "additionalProperties": false,
      "required": ["enabled", "revision"],
      "properties": {
        "enabled": {
          "type": "boolean"
        },
        "revision": {
          "type": "integer",
          "minimum": 1
        },
        "variant_id": {
          "type": "string",
          "minLength": 1,
          "maxLength": 128,
          "pattern": "^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$"
        }
      },
      "allOf": [
        {
          "if": {
            "required": ["enabled"],
            "properties": { "enabled": { "const": true } }
          },
          "then": { "required": ["variant_id"] },
          "else": { "not": { "required": ["variant_id"] } }
        }
      ]
    },
