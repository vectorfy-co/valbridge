#!/usr/bin/env node
import { createAdapterCLI } from "@vectorfyco/valbridge-core";
import { convert } from "./index.js";

createAdapterCLI(convert);
