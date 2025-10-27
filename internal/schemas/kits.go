package schemas

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// KitModel represents a kit component with all its details
type KitModel struct {
	ID           types.String       `tfsdk:"id"`
	Type         types.String       `tfsdk:"type"`
	Name         types.String       `tfsdk:"name"`
	Description  types.String       `tfsdk:"description"`
	Version      types.String       `tfsdk:"version"`
	Requirements []RequirementModel `tfsdk:"requirements"`
}

// ToObjectValue converts KitModel to types.Object
func (k KitModel) ToObjectValue() (types.Object, error) {
	// Convert requirements to list value
	reqValues := make([]attr.Value, len(k.Requirements))
	for i, req := range k.Requirements {
		reqObj, err := req.ToObjectValue()
		if err != nil {
			return types.ObjectNull(KitModelType()), err
		}
		reqValues[i] = reqObj
	}

	reqList, diag := types.ListValue(types.ObjectType{AttrTypes: RequirementModelType()}, reqValues)
	if diag.HasError() {
		return types.ObjectNull(KitModelType()), fmt.Errorf("failed to create requirements list")
	}

	objVal, objDiag := types.ObjectValue(
		KitModelType(),
		map[string]attr.Value{
			"id":           k.ID,
			"type":         k.Type,
			"name":         k.Name,
			"description":  k.Description,
			"version":      k.Version,
			"requirements": reqList,
		},
	)
	if objDiag.HasError() {
		return types.ObjectNull(KitModelType()), fmt.Errorf("failed to create kit object")
	}
	return objVal, nil
}

// KitModelType returns the attribute types for KitModel
func KitModelType() map[string]attr.Type {
	return map[string]attr.Type{
		"id":           types.StringType,
		"type":         types.StringType,
		"name":         types.StringType,
		"description":  types.StringType,
		"version":      types.StringType,
		"requirements": types.ListType{ElemType: types.ObjectType{AttrTypes: RequirementModelType()}},
	}
}

// RequirementModelType returns the attribute types for RequirementModel
func RequirementModelType() map[string]attr.Type {
	return map[string]attr.Type{
		"name":          types.StringType,
		"instructions":  types.ListType{ElemType: types.ObjectType{AttrTypes: InstructionModelType()}},
		"verifications": types.ListType{ElemType: types.ObjectType{AttrTypes: VerificationModelType()}},
	}
}

// VerificationModelType returns the attribute types for VerificationModel
func VerificationModelType() map[string]attr.Type {
	return map[string]attr.Type{
		"command": types.StringType,
		"expect":  types.StringType,
	}
}

// InstructionModelType returns the attribute types for InstructionModel
func InstructionModelType() map[string]attr.Type {
	return map[string]attr.Type{
		"prompt":      types.StringType,
		"constraints": types.ListType{ElemType: types.StringType},
	}
}

// ToObjectValue converts RequirementModel to types.Object
func (r RequirementModel) ToObjectValue() (types.Object, error) {
	// Convert instructions
	var instructionValue attr.Value
	if len(r.Instructions) > 0 {
		instValues := []attr.Value{}
		for _, inst := range r.Instructions {
			instObj, err := inst.ToObjectValue()
			if err != nil {
				return types.ObjectNull(RequirementModelType()), err
			}
			instValues = append(instValues, instObj)
		}
		instList, _ := types.ListValue(types.ObjectType{AttrTypes: InstructionModelType()}, instValues)
		instructionValue = instList
	} else {
		instructionValue = types.ListNull(types.ObjectType{AttrTypes: InstructionModelType()})
	}

	// Convert verifications if present
	var verificationValue attr.Value
	if len(r.Verifications) > 0 {
		verValues := []attr.Value{}
		for _, v := range r.Verifications {
			verObj, err := v.ToObjectValue()
			if err != nil {
				return types.ObjectNull(RequirementModelType()), err
			}
			verValues = append(verValues, verObj)
		}
		verList, _ := types.ListValue(types.ObjectType{AttrTypes: VerificationModelType()}, verValues)
		verificationValue = verList
	} else {
		verificationValue = types.ListNull(types.ObjectType{AttrTypes: VerificationModelType()})
	}

	objVal, objDiag := types.ObjectValue(
		RequirementModelType(),
		map[string]attr.Value{
			"name":          r.Name,
			"instructions":  instructionValue,
			"verifications": verificationValue,
		},
	)
	if objDiag.HasError() {
		return types.ObjectNull(RequirementModelType()), fmt.Errorf("failed to create requirement object")
	}
	return objVal, nil
}

// ToObjectValue converts VerificationModel to types.Object
func (v VerificationModel) ToObjectValue() (types.Object, error) {
	objVal, objDiag := types.ObjectValue(
		VerificationModelType(),
		map[string]attr.Value{
			"command": v.Command,
			"expect":  v.Expect,
		},
	)
	if objDiag.HasError() {
		return types.ObjectNull(VerificationModelType()), fmt.Errorf("failed to create verification object")
	}
	return objVal, nil
}

// ToObjectValue converts InstructionModel to types.Object
func (i InstructionModel) ToObjectValue() (types.Object, error) {
	// Convert constraints to list
	var constraintsValue attr.Value
	if i.Constraints != nil && len(i.Constraints) > 0 {
		constraintValues := make([]attr.Value, len(i.Constraints))
		for idx, c := range i.Constraints {
			constraintValues[idx] = c
		}
		constraintsList, _ := types.ListValue(types.StringType, constraintValues)
		constraintsValue = constraintsList
	} else {
		constraintsValue = types.ListNull(types.StringType)
	}

	objVal, objDiag := types.ObjectValue(
		InstructionModelType(),
		map[string]attr.Value{
			"prompt":      i.Prompt,
			"constraints": constraintsValue,
		},
	)
	if objDiag.HasError() {
		return types.ObjectNull(InstructionModelType()), fmt.Errorf("failed to create instruction object")
	}
	return objVal, nil
}
