package apis

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type vmConsoleTicketResponse struct {
	Ticket    string `json:"ticket"`
	ExpiresAt string `json:"expiresAt"`
}

// vmConsoleTicketHandler issues a short-lived WebSocket console ticket.
// deps provides auth, user lookup, VM lookup, and the signed ticket service.
// The route parameter is the VM name in the current user's namespace.
// This handler is used by the frontend before opening a browser WebSocket.
func vmConsoleTicketHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c, deps)
		if !ok {
			return
		}

		vm, err := vmServiceFromDependencies(deps).Get(c.Request.Context(), user.Namespace, c.Param("name"))
		if err != nil {
			writeVMError(c, err, "failed to read virtual machine")
			return
		}
		if !vmConsoleAllowed(vm.Phase, vm.Delete) {
			c.JSON(http.StatusConflict, gin.H{"message": "console is available only while the virtual machine is running"})
			return
		}

		token, ticket, err := deps.ConsoleTickets.Issue(user.Username, user.Namespace, vm.Name, time.Now().UTC())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to issue console ticket"})
			return
		}

		c.JSON(http.StatusOK, vmConsoleTicketResponse{
			Ticket:    token,
			ExpiresAt: ticket.ExpiresAt.Format(time.RFC3339),
		})
	}
}

// vmConsoleHandler upgrades a browser request into a VM serial console WebSocket.
// deps provides the signed ticket verifier, VM lookup, and KubeVirt console connector.
// The route parameter is the VM name the ticket must target.
// This handler is used by the dedicated VM console page.
func vmConsoleHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps.ConsoleConnector == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "console connector is not configured"})
			return
		}

		namespace, ok := consumeConsoleTicket(c, deps)
		if !ok {
			return
		}

		vm, err := vmServiceFromDependencies(deps).Get(c.Request.Context(), namespace, c.Param("name"))
		if err != nil {
			writeVMError(c, err, "failed to read virtual machine")
			return
		}
		if !vmConsoleAllowed(vm.Phase, vm.Delete) {
			c.JSON(http.StatusConflict, gin.H{"message": "console is available only while the virtual machine is running"})
			return
		}

		upstream, err := deps.ConsoleConnector.Connect(c.Request.Context(), namespace, c.Param("name"))
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"message": "failed to connect virtual machine console"})
			return
		}

		browser, err := vmConsoleUpgrader().Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			_ = upstream.Close()
			return
		}

		_ = bridgeConsole(c.Request.Context(), browser, upstream)
	}
}

// consumeConsoleTicket validates the WebSocket ticket query parameter.
// c carries the route VM name and ticket query value from the browser request.
// deps provides the signed ticket service shared by console handlers.
// The returned namespace is the VM namespace embedded in the verified ticket.
func consumeConsoleTicket(c *gin.Context, deps Dependencies) (string, bool) {
	token := c.Query("ticket")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "console ticket is required"})
		return "", false
	}

	ticket, err := deps.ConsoleTickets.Consume(token, "", c.Param("name"), time.Now().UTC())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "console ticket is invalid or expired"})
		return "", false
	}

	return ticket.Namespace, true
}

// vmConsoleAllowed reports whether a VM status can expose a console.
// phase is the observed KiteVirtualMachine status phase.
// deleting is the requested delete flag from the VM spec.
// This function is used before issuing and accepting console tickets.
func vmConsoleAllowed(phase string, deleting bool) bool {
	return phase == "Running" && !deleting
}
