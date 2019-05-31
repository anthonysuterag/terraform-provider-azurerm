package azurerm

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2015-04-08/documentdb"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/response"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/validate"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmCosmosDbSQLContainer() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmCosmosDbSQLContainerCreateUpdate,
		Read:   resourceArmCosmosDbSQLContainerRead,
		Delete: resourceArmCosmosDbSQLContainerDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.CosmosEntityName,
			},

			"resource_group_name": resourceGroupNameSchema(),

			"account_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.CosmosAccountName,
			},

			"database_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.CosmosEntityName,
			},
		},
	}
}

func resourceArmCosmosDbSQLContainerCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).cosmosAccountsClient
	ctx := meta.(*ArmClient).StopContext

	name := d.Get("name").(string)
	resourceGroup := d.Get("resource_group_name").(string)
	account := d.Get("account_name").(string)
	database := d.Get("database_name").(string)

	if requireResourcesToBeImported && d.IsNewResource() {
		existing, err := client.GetSQLContainer(ctx, resourceGroup, account, database, name)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of creating Cosmos SQL Container %s (Account %s / Database %s): %+v", name, account, database, err)
			}
		} else {
			id, err := azure.CosmosGetIDFromResponse(existing.Response)
			if err != nil {
				return fmt.Errorf("Error generating import ID for Cosmos SQL Container '%s' (Account %s / Database %s)", name, account, database)
			}
			return tf.ImportAsExistsError("azurerm_cosmosdb_sql_container", id)
		}
	}

	parameters := documentdb.SQLContainerCreateUpdateParameters{
		SQLContainerCreateUpdateProperties: &documentdb.SQLContainerCreateUpdateProperties{
			Resource: &documentdb.SQLContainerResource{
				ID: &name,
			},
			Options: map[string]*string{},
		},
	}

	future, err := client.CreateUpdateSQLContainer(ctx, resourceGroup, account, database, name, parameters)
	if err != nil {
		return fmt.Errorf("Error issuing create/update request for Cosmos SQL Container %s (Account %s / Database %s): %+v", name, account, database, err)
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting on create/update future for Cosmos SQL Container %s (Account %s / Database %s): %+v", name, account, database, err)
	}

	resp, err := client.GetSQLContainer(ctx, resourceGroup, account, database, name)
	if err != nil {
		return fmt.Errorf("Error making get request for Cosmos SQL Container %s (Account %s / Database %s): %+v", name, account, database, err)
	}
	id, err := azure.CosmosGetIDFromResponse(resp.Response)
	if err != nil {
		return fmt.Errorf("Error retrieving the ID for Cosmos SQL Container %s (Account %s / Database %s) ID: %v", name, account, database, err)
	}
	d.SetId(id)

	return resourceArmCosmosDbSQLContainerRead(d, meta)
}

func resourceArmCosmosDbSQLContainerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).cosmosAccountsClient
	ctx := meta.(*ArmClient).StopContext

	id, err := azure.ParseCosmosDatabaseContainerID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.GetSQLContainer(ctx, id.ResourceGroup, id.Account, id.Database, id.Container)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[INFO] Error reading Cosmos SQL Container %s (Account %s / Database %s) - removing from state", id.Container, id.Account, id.Database)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error reading Cosmos SQL Container %s (Account %s / Database %s): %+v", id.Container, id.Account, id.Database, err)
	}

	d.Set("resource_group_name", id.ResourceGroup)
	d.Set("account_name", id.Account)
	d.Set("database_name", id.Database)
	if properties := resp.SQLContainerProperties; properties != nil {
		d.Set("name", properties.ID)
	}

	return nil
}

func resourceArmCosmosDbSQLContainerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).cosmosAccountsClient
	ctx := meta.(*ArmClient).StopContext

	id, err := azure.ParseCosmosDatabaseContainerID(d.Id())
	if err != nil {
		return err
	}

	future, err := client.DeleteSQLContainer(ctx, id.ResourceGroup, id.Account, id.Database, id.Container)
	if err != nil {
		if !response.WasNotFound(future.Response()) {
			return fmt.Errorf("Error deleting Cosmos SQL Database %s (Account %s): %+v", id.Database, id.Account, err)
		}
	}

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err != nil {
		return fmt.Errorf("Error waiting on delete future for Cosmos SQL Database %s (Account %s): %+v", id.Database, id.Account, err)
	}

	return nil
}