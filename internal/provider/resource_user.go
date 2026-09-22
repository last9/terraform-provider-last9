package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

var userEmailRE = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func resourceUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceUserCreate,
		ReadContext:   resourceUserRead,
		UpdateContext: resourceUserUpdate,
		DeleteContext: resourceUserDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceUserImport,
		},
		Schema: map[string]*schema.Schema{
			"email": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "User email address (invite identity)",
				ValidateFunc: validation.StringMatch(userEmailRE, "must be a valid email address"),
			},
			"role": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ValidateFunc: validation.StringInSlice([]string{
					"admin", "editor", "viewer",
				}, false),
				Description: "Organization role: admin, editor, or viewer. Omit to use the org default on first login.",
			},
			"active": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the user is active. Setting false soft-deletes/deactivates the user.",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Display name (populated after the user accepts the invite)",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User status (e.g. invited, active)",
			},
			"organization_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceUserCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	email := strings.TrimSpace(d.Get("email").(string))
	role := d.Get("role").(string)

	if err := c.InviteUsers([]string{email}, role); err != nil {
		return diag.FromErr(fmt.Errorf("invite user: %w", err))
	}

	// Invited users can take a moment to appear in list; retry briefly.
	var user *client.User
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		user, err = c.FindUserByEmail(email)
		if err == nil {
			break
		}
		time.Sleep(time.Duration(attempt+1) * 300 * time.Millisecond)
	}
	if err != nil {
		return diag.FromErr(fmt.Errorf("invite succeeded but could not resolve user: %w", err))
	}
	d.SetId(user.ID)

	// If role was set but list doesn't reflect it yet, force-set via roles API.
	if role != "" && !strings.EqualFold(user.Role, role) {
		if err := c.UpdateUserRole(user.ID, role); err != nil {
			return diag.FromErr(fmt.Errorf("set user role after invite: %w", err))
		}
	}

	if !d.Get("active").(bool) {
		if err := c.PatchUser(user.ID, false); err != nil {
			return diag.FromErr(fmt.Errorf("deactivate invited user: %w", err))
		}
	}

	return resourceUserRead(ctx, d, m)
}

func resourceUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)

	user, err := c.FindUserByID(d.Id())
	if err != nil {
		if isNotFoundError(err) || strings.Contains(err.Error(), "not found") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("read user: %w", err))
	}

	_ = d.Set("email", user.Email)
	_ = d.Set("name", user.Name)
	_ = d.Set("status", user.Status)
	_ = d.Set("organization_id", user.OrganizationID)
	_ = d.Set("active", user.DeletedAt == nil)

	role := user.Role
	if role == "" {
		if r, rerr := c.GetUserRole(user.ID); rerr == nil {
			role = r
		}
	}
	if role != "" {
		_ = d.Set("role", role)
	}
	return nil
}

func resourceUserUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)

	if d.HasChange("role") {
		role := d.Get("role").(string)
		if role == "" {
			return diag.Errorf("role cannot be cleared; set admin, editor, or viewer")
		}
		if err := c.UpdateUserRole(d.Id(), role); err != nil {
			return diag.FromErr(fmt.Errorf("update user role: %w", err))
		}
	}

	if d.HasChange("active") {
		active := d.Get("active").(bool)
		if err := c.PatchUser(d.Id(), active); err != nil {
			return diag.FromErr(fmt.Errorf("update user active: %w", err))
		}
	}

	return resourceUserRead(ctx, d, m)
}

func resourceUserDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*client.Client)
	if err := c.DeleteUser(d.Id()); err != nil {
		return diag.FromErr(fmt.Errorf("delete user: %w", err))
	}
	d.SetId("")
	return nil
}

func resourceUserImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	c := m.(*client.Client)
	id := d.Id()

	// Allow import by email or user ID.
	if strings.Contains(id, "@") {
		user, err := c.FindUserByEmail(id)
		if err != nil {
			return nil, err
		}
		d.SetId(user.ID)
		_ = d.Set("email", user.Email)
	} else {
		user, err := c.FindUserByID(id)
		if err != nil {
			return nil, err
		}
		_ = d.Set("email", user.Email)
	}
	return []*schema.ResourceData{d}, nil
}
