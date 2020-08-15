package com.myorg;

import software.amazon.awscdk.core.App;

public class ExpenseHandlingWorkflowApp {
    public static void main(final String[] args) {
        App app = new App();

        new ExpenseHandlingWorkflowStack(app, "ExpenseHandlingWorkflowStack");

        app.synth();
    }
}
