package tenant

import (
	"context"
	"fmt"
	"os"

	"github.com/projectsyn/lieutenant-operator/pkg/apis"
	synv1alpha1 "github.com/projectsyn/lieutenant-operator/pkg/apis/syn/v1alpha1"
	"github.com/projectsyn/lieutenant-operator/pkg/git/manager"
	"github.com/projectsyn/lieutenant-operator/pkg/pipeline"
	roleUtil "github.com/projectsyn/lieutenant-operator/pkg/role"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// CommonClassName is the name of the tenant's common class
	CommonClassName         = "common"
	DefaultGlobalGitRepoURL = "DEFAULT_GLOBAL_GIT_REPO_URL"
	ClusterClassContent     = `classes:
- %s.%s
`
)

func addDefaultClassFile(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	commonClassFile := CommonClassName + ".yml"
	if obj.GetGitTemplate().TemplateFiles == nil {
		obj.GetGitTemplate().TemplateFiles = map[string]string{}
	}
	if _, ok := obj.GetGitTemplate().TemplateFiles[commonClassFile]; !ok {
		obj.GetGitTemplate().TemplateFiles[commonClassFile] = ""
	}
	return pipeline.Result{}
}

func updateTenantGitRepo(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	tenantCR, ok := obj.(*synv1alpha1.Tenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	var oldFiles map[string]string
	if tenantCR.Spec.GitRepoTemplate != nil {
		oldFiles = tenantCR.Spec.GitRepoTemplate.TemplateFiles
	} else {
		tenantCR.Spec.GitRepoTemplate = &synv1alpha1.GitRepoTemplate{}
	}

	tenantCR.Spec.GitRepoTemplate.TemplateFiles = map[string]string{}

	clusterList := &synv1alpha1.ClusterList{}

	selector := labels.Set(map[string]string{apis.LabelNameTenant: tenantCR.Name}).AsSelector()

	listOptions := &client.ListOptions{
		LabelSelector: selector,
		Namespace:     obj.GetObjectMeta().GetNamespace(),
	}

	err := data.Client.List(context.TODO(), clusterList, listOptions)
	if err != nil {
		return pipeline.Result{Err: err}
	}

	for _, cluster := range clusterList.Items {
		fileName := cluster.GetName() + ".yml"
		fileContent := fmt.Sprintf(ClusterClassContent, tenantCR.Name, CommonClassName)
		tenantCR.Spec.GitRepoTemplate.TemplateFiles[fileName] = fileContent
		delete(oldFiles, fileName)
	}

	for fileName := range oldFiles {
		if fileName == CommonClassName+".yml" {
			tenantCR.Spec.GitRepoTemplate.TemplateFiles[CommonClassName+".yml"] = ""
		} else {
			tenantCR.Spec.GitRepoTemplate.TemplateFiles[fileName] = manager.DeletionMagicString

		}
	}

	return pipeline.Result{}
}

func setGlobalGitRepoURL(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	instance, ok := obj.(*synv1alpha1.Tenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	defaultGlobalGitRepoURL := os.Getenv(DefaultGlobalGitRepoURL)
	if len(instance.Spec.GlobalGitRepoURL) == 0 && len(defaultGlobalGitRepoURL) > 0 {
		instance.Spec.GlobalGitRepoURL = defaultGlobalGitRepoURL
	}
	return pipeline.Result{}
}

func applyTemplateFromTenantTemplate(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	tenant, ok := obj.(*synv1alpha1.Tenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	key := types.NamespacedName{Name: "default", Namespace: obj.GetObjectMeta().GetNamespace()}
	template := &synv1alpha1.TenantTemplate{}
	if err := data.Client.Get(context.TODO(), key, template); err != nil {
		if errors.IsNotFound(err) || runtime.IsNotRegisteredError(err) {
			// The absence of a template is not an error.
			// It simply means that there is nothing to do.
			data.Log.Info("No template found to apply to tenant.")
			return pipeline.Result{}
		}
		return pipeline.Result{
			Err:     err,
			Requeue: true,
		}
	}

	if err := tenant.ApplyTemplate(template); err != nil {
		return pipeline.Result{Err: err}
	}

	return pipeline.Result{}
}

func createServiceAccount(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	tenant, ok := obj.(*synv1alpha1.Tenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	r, ok := data.Reconciler.(*ReconcileTenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	sa, err := r.NewServiceAccount(tenant)
	if err != nil {
		return pipeline.Result{Err: fmt.Errorf("failed to create ServiceAccount for tenant: %w", err)}
	}

	err = data.Client.Create(context.TODO(), sa)
	if err != nil && !errors.IsAlreadyExists(err) {
		return pipeline.Result{Err: err}
	}

	return pipeline.Result{}
}

func createRole(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	tenant, ok := obj.(*synv1alpha1.Tenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	r, ok := data.Reconciler.(*ReconcileTenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	role, err := r.NewRole(tenant)
	if err != nil {
		return pipeline.Result{Err: fmt.Errorf("failed to create Role for tenant: %w", err)}
	}

	err = data.Client.Create(context.TODO(), role)
	if err != nil && !errors.IsAlreadyExists(err) {
		return pipeline.Result{Err: err}
	}

	return pipeline.Result{}
}

func tenantUpdateRole(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	tenant, ok := obj.(*synv1alpha1.Tenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	name := types.NamespacedName{Name: tenant.Name, Namespace: tenant.Namespace}
	role := &rbacv1.Role{}
	if err := data.Client.Get(context.TODO(), name, role); err != nil {
		return pipeline.Result{Err: fmt.Errorf("failed to get role for tenant: %v", err)}
	}

	if !roleUtil.AddResourceNames(role, tenant.Name) {
		return pipeline.Result{}
	}

	if err := data.Client.Update(context.TODO(), role); err != nil {
		return pipeline.Result{Err: fmt.Errorf("failed to update role for tenant: %v", err)}
	}

	return pipeline.Result{}
}

func createRoleBinding(obj pipeline.Object, data *pipeline.Context) pipeline.Result {
	tenant, ok := obj.(*synv1alpha1.Tenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	r, ok := data.Reconciler.(*ReconcileTenant)
	if !ok {
		return pipeline.Result{Err: fmt.Errorf("object is not a tenant")}
	}

	binding, err := r.NewRoleBinding(tenant)
	if err != nil {
		return pipeline.Result{Err: fmt.Errorf("failed to create RoleBinding for tenant: %w", err)}
	}

	err = data.Client.Create(context.TODO(), binding)
	if err != nil && !errors.IsAlreadyExists(err) {
		return pipeline.Result{Err: err}
	}

	return pipeline.Result{}
}
