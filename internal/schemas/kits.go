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
		"name":         types.StringType,
		"instructions": types.ListType{ElemType: types.StringType},
		"verification": types.ObjectType{AttrTypes: VerificationModelType()},
	}
}

// VerificationModelType returns the attribute types for VerificationModel
func VerificationModelType() map[string]attr.Type {
	return map[string]attr.Type{
		"command": types.StringType,
		"expect":  types.StringType,
	}
}

// ToObjectValue converts RequirementModel to types.Object
func (r RequirementModel) ToObjectValue() (types.Object, error) {
	// Convert instructions
	instValues := make([]attr.Value, len(r.Instructions))
	for i, inst := range r.Instructions {
		instValues[i] = inst
	}
	instList, _ := types.ListValue(types.StringType, instValues)

	// Convert verification if present
	var verificationValue attr.Value
	if r.Verification != nil {
		verObj, err := r.Verification.ToObjectValue()
		if err != nil {
			return types.ObjectNull(RequirementModelType()), err
		}
		verificationValue = verObj
	} else {
		verificationValue = types.ObjectNull(VerificationModelType())
	}

	objVal, objDiag := types.ObjectValue(
		RequirementModelType(),
		map[string]attr.Value{
			"name":         r.Name,
			"instructions": instList,
			"verification": verificationValue,
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
